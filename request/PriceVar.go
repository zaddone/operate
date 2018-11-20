package request
import(
	"math"
	"sync"
	//"time"
	"net/url"
	"fmt"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/oanda"
	"log"
	"io"
	"bufio"
	"encoding/json"
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
func (self *PriceVar) RunPriceStream(){
	var err error
	var r []byte
	var p bool
	for{
		err = clientHttp(0,
		"GET",
		config.Conf.GetStreamAccPath()+"/pricing/stream?"+url.Values{"instruments":[]string{self.Ins.Name}}.Encode(),
		nil,
		func(statusCode int,data io.Reader) error {
			if statusCode != 200 {
				return fmt.Errorf("%d",statusCode)
			}
			buf := bufio.NewReader(data)
			for{
				r,p,err = buf.ReadLine()
				if p {
					return fmt.Errorf("%s",string(r))
				}
				if len(r)>0 {
					var d oanda.Price
					if er := json.Unmarshal(r,&d);er != nil {
						log.Println(er,string(r))
						continue
					}
					name := string(d.Instrument)
					if name == "" {
						continue
					}
					go self.AddPrice(&d)
				}
				if err != nil {
					if err != io.EOF {
						//panic(err)
						log.Println("line",err)
					}
					return err
				}
			}
			return nil
		})
		if err != nil {
			//panic(err)
			fmt.Println(err)
		}
	}

}
func (self *PriceVar)GetLastPr() *oanda.Price {
	self.RLock()
	defer self.RUnlock()
	//if self.last ==nil {
	//	return nil
	//}
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
	//go p.RunPriceStream()
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
	//defer func(){
	//	self.list = append(self.list,v)
	//}()

	self.list = append(self.list,v)
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
	difp :=sumdif/float64(le)*1.2
	if (maxid == 0) || (math.Abs(Max) < difp) {
		return
	}
	//fmt.Println(self.Ins.Name,Max,maxid)
	self.last = &PriceVar{Ins:self.Ins,list:self.list[:maxid],dis:self.dis}
	self.list = self.list[maxid:]
	self.dis = Max
	gr := config.GetGran(self.TimeInterval())
	if gr == nil {
		return
	}
	canc,err := NewCanCache(self.Ins.Name,gr,100)
	if err != nil {
		fmt.Println(err)
		return
	}
	le = len(canc.nodes)
	if le < 3 {
		return
	}
	//timediff := v.Time.Time() - canc.cans[len(canc.cans)-1].Time
	//if timediff<0 {
	//	//fmt.Println("timediff",timediff)
	//	return
	//}
	_sl :=canc.nodes[le-1]
	_tp :=canc.nodes[le-2]
	//if (_sl.dis >0) != (_tp.dis>0) {
	//	return
	//}
	sl := _sl.ca.getVal()
	tp := _tp.ca.getVal()
	df := tp - sl
	_df := (df>0)
	if _df != (Max>0) {
		return
	}
	if math.Abs(df) < difp*5 {
		return
	}
	gr_2 := config.GetGran(canc.cans[len(canc.cans)-1].Time - _tp.ca.Time)
	if gr_2 == nil {
		return
	}
	//var pric,sl,tp float64
	//if _df {
	//	pric = v.Bid()
	//}else{
	//	pric = v.Ask()
	//}

	//__sl := sl - pric
	//__tp := tp - pric
	//hk := ((_sl>0) == (_tp>0))
	//if hk {
	//	return
	//}
	//difp = difp*3
	//if difp < self.Ins.MinimumTrailingStopDistance ||
	//	math.Abs(_sl) < difp ||
	//	math.Abs(_tp) < difp {
	//	return
	//}

	canc_2,err := NewCanCache(self.Ins.Name,gr_2,10)
	if err != nil {
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
