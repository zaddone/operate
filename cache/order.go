package cache
import(
	"github.com/zaddone/operate/oanda"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/request"
	"os"
	"sync"
	"path/filepath"
	"math"
	"fmt"
	"log"

)
type order struct{

	e element
	tp float64
	sl float64
	orderId string
	ins *oanda.Instrument
	f *os.File
	sync.Mutex
	le *level
	k float64
	path string

}
func NewOrder(e element,ins *oanda.Instrument,le *level,tp,sl float64) (o *order) {

	o = &order{
		e:e,
		ins:ins,
		le:le,
		tp:tp,
		sl:sl,
		k:math.Pow10(int(ins.DisplayPrecision)),
	}
	le.lastOrder = o
	o.openFile()
	go o.load()
	go o.postOrder()
	return o

}
func (self *order) close(){
	old := self.f.Name()
	self.f.Close()
	if err := os.Rename(old,old+".log"); err != nil {
		log.Println(err)
	}
}
func (self *order) openFile(){

	self.path = filepath.Join(
		config.Conf.LogPath,
		request.GetNowTime(self.e.DateTime()).Format("20060102"),
	)
	_,err := os.Stat(self.path)
	if err != nil {
		err = os.MkdirAll(self.path,0700)
		if err != nil {
			panic(err)
		}
	}
	self.f,err = os.OpenFile(
		filepath.Join(
			self.path,
			fmt.Sprintf("%s_%d_%d_%d",
				self.ins.Name,
				self.le.tag,
				int(self.tp*self.k),
				int(self.sl*self.k),
			),
		),
		os.O_APPEND|os.O_CREATE|os.O_RDWR|os.O_SYNC,
		0700,
	)

}
func (self *order) save(e element){

	self.Lock()
	self.f.WriteString(fmt.Sprintf("%d %d %d\n",int(self.k*e.Middle()),int(self.k*e.Diff()),e.DateTime()))

	self.Unlock()
}
func (self *order) load(){
	self.Lock()
	for _,li := range self.le.list{
		li.Read(func(_e interface{}){
			__e := _e.(element)
			self.f.WriteString(fmt.Sprintf("%d %d %d\n",int(self.k*__e.Middle()),int(self.k*__e.Diff()),__e.DateTime()))
		})
	}
	self.f.WriteString("end\n")
	self.Unlock()

}
func (self *order) postOrder() {

	res,err := request.HandleOrder(
		self.ins.Name,
		int((request.MarginRate*float64(config.Conf.Units))/self.e.Middle()),
		"",
		self.ins.StandardPrice(self.tp),
		self.ins.StandardPrice(self.sl),
	)
	if err == nil {
		self.orderId = string(res.OrderFillTransaction.Id)
	}

}
func (self *order) check(e element) (b bool) {
	go self.save(e)
	if self.tp>self.sl {
		if (e.Middle() > self.tp) || (e.Middle() < self.sl) {
			return
		}
	}else{
		if (e.Middle() < self.tp) || (e.Middle() > self.sl) {
			return
		}
	}
	b = true
	self.close()
	if self.orderId != "" && request.CheckTrades(self.orderId) {
		_,err := request.CloseTrades(self.orderId,"ALL")
		if err != nil {
			return
		}
		if request.CheckTrades(self.orderId) {
			return
		}
		_,err = request.ClosePosition(self.ins.Name,"ALL")
		if err != nil {
			return
		}
		if request.CheckTrades(self.orderId) {
			return
		}
	}
	return


}
