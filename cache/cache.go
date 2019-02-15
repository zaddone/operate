package cache
import(
	//"github.com/zaddone/operate/request"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/oanda"
	//"path/filepath"
	"sync"
	"math"
	"time"
	//"log"
	//"os"
	//"fmt"
)
const(
	TimeOut = 14400
)
type element interface{
	DateTime() int64
	Middle() float64
	Diff() float64
	Read(func(interface{}))
}

type eNode struct {
	li []element
	middle float64
	diff float64
}
func NewNode(li []element) (n *eNode) {
	le:=len(li)
	n = &eNode{
		li:li,
		diff:li[le-1].Middle() - li[0].Middle(),
	}
	for _,e := range li {
		n.middle+=e.Middle()
	}
	n.middle /= float64(len(li))
	return
}
func (self *eNode) Read(hand func(interface{})){
	for _,e := range self.li {
		e.Read(hand)
	}
}

func (self *eNode) DateTime() int64{
	le := len(self.li)
	if le == 0 {
		return 0
	}
	return self.li[0].DateTime()
}
func (self *eNode) Middle() float64{
	return self.middle
}
func (self *eNode) Diff() float64{
	return self.diff
}
type level struct {

	list []element
	dis float64
	par *level
	tag int

	max float64
	maxid int
	update bool
	//next *part
	tp element
	sl element

	lastOrder *order

}
func NewLevel(tag int) *level {
	return &level{
		list:make([]element,0,100),
		tag:tag,
	}
}
func (self *level) getBoundVal() (tp,sl element){
	if self.maxid == 0 {
		return self.tp,self.sl
	}
	return self.list[0],self.list[self.maxid]
}

func (self *level) checkOrder(e element,ins *oanda.Instrument,orderHandle func(*order)) bool {

	if self.par == nil {
		return false
	}
	if self.update  &&
	self.par.checkOrder(e,ins,orderHandle) {
		return true
	}
	tp,sl := self.getBoundVal()
	if (self.par.lastOrder != nil) &&
	(tp.Middle() == self.par.lastOrder.tp) {
		return false
	}
	tp_ :=math.Abs(tp.Middle() - e.Middle())
	sl_ :=math.Abs(sl.Middle() - e.Middle())
	if (tp_ < sl_) ||
	(sl_ < math.Abs(e.Diff())*config.Conf.Val) {
	//if (tp_ < sl_){
		return false
	}

	le := len(self.par.list)-1
	var sum,sum1,c,c1 float64
	for i:=0;i<le;i++{
		d :=self.par.list[i].Diff()
		if d > 0 {
			sum += d
			c++
		}else{
			sum1 += d
			c1++
		}
	}
	if c ==0 || c1==0 {
		return false
	}
	diff:= tp.Middle()-sl.Middle()
	if (diff > 0) == (sum > 0) {
		sum = math.Abs(sum/c)
		sum1 = math.Abs(sum1/c1)
		if (sum < sum1) || (math.Abs(diff) < sum) {
			return false
		}
	}else{
		sum = math.Abs(sum/c)
		sum1 = math.Abs(sum1/c1)
		if (sum > sum1) ||  (math.Abs(diff) <sum1) {
			return false
		}
	}
	orderHandle(NewOrder(
		e,
		ins,
		self.par,
		tp.Middle(),
		func()float64{
			if (tp.Middle()>sl.Middle()) {
				return sl.Middle()-math.Abs(sl.Diff())
			}
			return sl.Middle()+math.Abs(sl.Diff())
		}(),
	))
	return true

}

func (self *level) add(e element) {

	self.update = false
	le := len(self.list)
	if le == 0 {
		self.list = []element{e}
		return
	}

	self.list = append(self.list,e)
	var sumdif,absMax,diff,absDiff float64

	mid := e.Middle()
	self.maxid =0
	self.max = 0
	var _e element
	for i:=0 ; i<le ; i++ {
		_e = self.list[i]
		sumdif += math.Abs(_e.Diff())
		diff = mid - _e.Middle()
		if (diff>0) == (self.dis>0) {
			continue
		}
		absDiff = math.Abs(diff)
		if absDiff > absMax {
			self.maxid = i
			self.max = diff
			absMax = absDiff
		}
	}
	if (self.maxid == 0) || (absMax == 0) || (absMax < sumdif/float64(le)) {
		return
	}
	self.update = true
	if self.list[self.maxid].DateTime() - self.list[0].DateTime() > TimeOut {
		self.par = nil
	}else{
		if self.par == nil {
			self.par = NewLevel(self.tag+1)
		}
		self.par.add(NewNode(self.list[:self.maxid]))
	}

	self.tp = self.list[0]
	self.sl = self.list[self.maxid]
	self.list = self.list[self.maxid:]
	//self.list = make([]element,0,100)
	//copy(self.list,li)
	self.dis = self.max
	self.max = 0
	self.maxid = 0
	//return true

}

type Cache struct {

	Ins *oanda.Instrument
	part *level
	priceChan chan element
	stop chan bool
	sync.RWMutex
	//order *request.OrderInfo
	orders map[string]*order

}
func (self *Cache) GetLastElement() element {

	le := len(self.part.list)
	if le == 0 {
		return nil
	}
	return self.part.list[le-1]

}

func (self *Cache) EndTime() time.Time {
	le := len(self.part.list)
	if le == 0 {
		return time.Unix(0,0)
	}
	return time.Unix(self.part.list[le-1].DateTime(),0)
}
func NewCache(ins *oanda.Instrument) (c *Cache) {
	c = &Cache{
		part:NewLevel(0),
		Ins:ins,
		priceChan:make(chan element,10),
		stop:make(chan bool),
		//order:request.NewOrderInfo(ins),
		orders:make(map[string]*order),
	}
	go c.runAdd()
	return c
}
func (self *Cache) runAdd(){
	for{
		select{
		case <-self.stop:
			return
		case p:= <-self.priceChan:
			self.Lock()
			if e := self.GetLastElement(); (e!= nil) && ((p.DateTime() - e.DateTime()) >300) {
				self.part = NewLevel(0)
			}
			self.part.add(p)
			if self.part.update {
				self.part.checkOrder(p,self.Ins,func(o *order){
					self.orders[o.f.Name()] = o
				})
			}
			self.Unlock()
		}
	}
}
func (self *Cache) AddPrice(p element) {
	for k,o := range self.orders{
		go func(_o *order,_k string){
			if _o.check(p) {
				delete(self.orders,k)
			}
		}(o,k)
	}
	//go self.order.Check(p.Middle())
	self.priceChan<-p
}
