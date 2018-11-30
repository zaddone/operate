package cache
import(
	"github.com/zaddone/operate/request"
	"github.com/zaddone/operate/oanda"
	"sync"
	"math"
	"time"
)
const(
	MaxTag = 3
	TimeOut = 3600
)
type element interface{
	DateTime() int64
	Middle() float64
	Diff() float64
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
		diff:math.Abs(li[0].Middle() - li[le-1].Middle()),
	}
	for _,e := range li {
		n.middle+=e.Middle()
	}
	n.middle /= float64(len(li))
	return
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
	//tag int

	max float64
	maxid int
	update bool
	//next *part
	tp element
	sl element

}
func NewLevel() *level {
	return &level{
		list:make([]element,0,100),
		//tag:tag,
	}
}

func (self *level) checkOrder(orderHandle func(tp,sl float64,timeOut int64)) bool {

	if !self.update {
		return false
	}
	self.update = false
	if self.par == nil {
		return false
	}
	if self.par.checkOrder(orderHandle){
		return true
	}
	if (self.par.dis >0) != (self.dis>0) {
		return false
	}
	if self.par.max != 0 {
		return false
	}
	le := len(self.par.list)
	var sum float64
	for i:=0;i < le;i++{
		sum += self.par.list[i].Diff()
	}
	sum /= float64(le)
	tp := self.tp.Middle()
	sl := self.sl.Middle()
	if (tp>sl) {
		sl -=self.sl.Diff()
		tp +=self.tp.Diff()
	}else{
		sl +=self.sl.Diff()
		tp -=self.tp.Diff()
	}
	if math.Abs(tp - sl)<sum {
		return false
	}
	if math.Abs(self.dis)*2 > sum {
		return false
	}
	orderHandle(
		tp,
		sl,
		func(n int64)int64{
		if n<0 {
			return -n
		}
		return n
	}(self.tp.DateTime() - self.sl.DateTime())*2)
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
		sumdif += _e.Diff()
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
			self.par = NewLevel()
		}
		self.par.add(NewNode(self.list[:self.maxid]))
	}

	self.tp = self.list[0]
	self.sl = self.list[self.maxid]
	li := self.list[self.maxid:]
	self.list = make([]element,0,100)
	copy(self.list,li)
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
	order *request.OrderInfo

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
		part:NewLevel(),
		Ins:ins,
		priceChan:make(chan element),
		stop:make(chan bool),
		order:request.NewOrderInfo(ins),
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
			self.part.add(p)
			if self.part.update {
				//if self.part.par != nil {
					self.part.checkOrder(func(tp,sl float64,timeOut int64){
						//if (self.part.dis >0) == ((tp-sl) >0) {
						go self.order.UpdateNew(tp,sl,p.Middle(),timeOut)
						//}
					})
				//}
			}
			self.Unlock()
		}
	}
}
func (self *Cache) AddPrice(p element){
	go self.order.Check(p.Middle(),p.DateTime())
	self.priceChan<-p
}
