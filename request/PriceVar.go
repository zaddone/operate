package request
import(
	"math"
	"sync"
	//"time"
	"fmt"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/oanda"
	//"log"
)

type PriceVar struct {

	Ins *Instrument
	dis float64
	list []*oanda.Price
	last *PriceVar
	//CacheMap map[string]*CanCache
	order *OrderInfo
	priceChan chan *oanda.Price
	stop chan bool
	sync.RWMutex

	Pl  float64
	CountOrder int

}
func (self *PriceVar)GetLastPr() *oanda.Price {
	self.RLock()
	defer self.RUnlock()
	if self.last ==nil {
		return nil
	}
	le := len(self.list)
	if le == 0 {
		return nil
	}
	return self.list[le-1]

}
func NewPriceVar(ins *Instrument) (p *PriceVar) {
	p = &PriceVar{
		Ins:ins,
		priceChan:make(chan *oanda.Price,100),
		stop:make(chan bool),
	}
	go p.RunAdd()
	return
}
func (self *PriceVar) UpdateOrder() {

}
func (self *PriceVar) AddPrice(p *oanda.Price){
	self.priceChan<-p
}
func (self *PriceVar) RunAdd(){
	for{
		select{
		case <-self.stop:
			return
		case p:= <-self.priceChan:
			self.Lock()
			self.add(p)
			self.Unlock()
		}
	}
}

func (self *PriceVar) add(v *oanda.Price) {

	le := len(self.list)
	if le < 2 {
		self.list = append(self.list,v)
		return
	}
	if (v.Time.Time() - self.list[le-1].Time.Time()) > 60{
		self.list = []*oanda.Price{v}
		self.last = nil
		return
	}
	defer func(){
		self.list = append(self.list,v)
	}()

	var sumdif,Max,diff float64
	maxid := 0
	mid := v.Middle()
	//fmt.Println(mid)
	var _p *oanda.Price
	for i:=0 ; i<le ; i++ {
		_p= self.list[i]
		sumdif += _p.Diff()
		diff = mid - _p.Middle()
		if (diff>0) == (self.dis>0) {
			continue
		}
		if math.Abs(diff) > math.Abs(Max) {
			maxid = i
			Max = diff
		}
	}
	difp :=sumdif/float64(le)
	if (maxid == 0) || (math.Abs(Max) < difp) {
		return
	}
	//fmt.Println(self.Ins.Name,Max,maxid)
	self.last = &PriceVar{Ins:self.Ins,list:self.list[:maxid],dis:self.dis}
	self.list = append(self.list[maxid:],v)
	self.dis = Max
	gr := config.GetGran(self.TimeInterval())
	if gr == nil {
		return
	}
	canc,err := NewCanCache(self.Ins.Name,gr,500)
	if err != nil {
		fmt.Println(err)
		return
	}
	le = len(canc.nodes)
	if le < 2 {
		return
	}
	timediff := v.Time.Time() - canc.cans[len(canc.cans)-1].Time
	if timediff<0 {
		//fmt.Println("timediff",timediff)
		return
	}
	k1 :=canc.nodes[le-1]
	k2 :=canc.nodes[le-2]
	df := k2.ca.getVal() - k1.ca.getVal()
	_df := (df>0)
	if _df != (Max>0) {
		return
	}

	gr_2 := config.GetGran(v.Time.Time() - k2.ca.Time)
	if gr_2 == nil {
		return
	}
	var pric,sl,tp float64
	if _df {
		pric = v.Bid()
	}else{
		pric = v.Ask()
	}
	sl = k1.ca.getVal() - pric
	tp = k2.ca.getVal() - pric
	hk := ((sl>0) ==(tp>0))
	if hk {
		return
	}
	difp = difp*3
	if difp < self.Ins.MinimumTrailingStopDistance ||
		math.Abs(sl) < difp ||
		math.Abs(tp) < difp {
		return
	}
	canc_2,err := NewCanCache(self.Ins.Name,gr_2,100)
	if err != nil {
		fmt.Println(err)
		return
	}
	if (canc_2.endMax>0) != _df{
		return
	}
	unit := config.Conf.Units
	if !_df {
		unit = -unit
	}
	if self.order == nil {
		self.order = &OrderInfo{ins:self.Ins,tp:tp,sl:sl}
		self.order.PostNew()
		return
	}
	if self.order.GetResId() == "" {
		self.order.sl = sl
		self.order.tp = tp
		self.order.PostNew()
		return
	}
	if ((self.order.tp - self.order.sl)>0) != _df {
		_diff:=math.Abs(self.order.tp - self.order.sl)
		isUpdate:= false
		if math.Abs(tp - self.order.sl) > _diff {
			isUpdate =true
			self.order.tp = tp
		}
		if math.Abs(self.order.tp - sl) > _diff {
			isUpdate =true
			self.order.sl = sl
		}
		if isUpdate {
			self.order.Update()
		}
	}else{
		if math.Abs(df) > math.Abs(self.order.tp - self.order.sl) {
			self.order.Close()
			self.order.sl = sl
			self.order.tp = tp
			self.order.PostNew()
		}
	}

}

func (self *PriceVar) show() float64 {

	if self.last == nil {
		return 0
	}
	return float64(self.last.TimeInterval())
	//return math.Pow(10,self.Ins.DisplayPrecision) * (self.list[len(self.list)-1].middle() - self.list[0].middle())

}

func (self *PriceVar) PriceInterval() float64 {

	self.RLock()
	defer self.RUnlock()
	le := len(self.list)
	if le == 0 {
		return 0
	}
	return self.list[le-1].Middle() - self.list[0].Middle()

}

func (self *PriceVar) TimeInterval () int64 {

	le := len(self.list)
	if le == 0 {
		return 0
	}
	return self.list[le-1].Time.Time() - self.list[0].Time.Time()

}
