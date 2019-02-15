package request
import(
	"io/ioutil"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/oanda"
	//"log"
	"encoding/json"
	"net/url"
	"io"
	"fmt"
	//"time"
	"strconv"
	"log"
	//"math"
)
func CheckTrades(trid string)bool {
	res,err := GetTrades(trid)
	if err != nil {
		return false
	}
	ct := res["trade"].(map[string]interface{})["closeTime"]
	if ct != nil {
		//fmt.Println("close Time",ct)
		return true
	}
	return false

}

func ListTrades() (res map[string]interface{},err error) {
	res = map[string]interface{}{}
	err = ClientHttp(0,"GET",
	config.Conf.GetAccPath() + "/openTrades",nil,
	func(statusCode int,body io.Reader) error{
		if statusCode != 200 {
			msg,_ := ioutil.ReadAll(body)
			return fmt.Errorf("%v",string(msg))
		}
		if er :=  json.NewDecoder(body).Decode(&res) ;er != nil {
			return er
		}
		return nil
	})
	return
}
func CloseAllTrades() error {
	res,err := ListTrades()
	if err != nil {
		return err
	}
	for _,tr := range res["trades"].([]interface{}){
		_,err = CloseTrades(tr.(map[string]interface{})["id"].(string),"ALL")
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func GetTrades(id string) (res map[string]interface{},err error){

	res = map[string]interface{}{}
	err = clientHttp(0,"GET",
	config.Conf.GetAccPath() + "/trades/"+id,nil,
	func(statusCode int,body io.Reader) (er error){
		if statusCode != 200 {
			msg,_ := ioutil.ReadAll(body)
			return fmt.Errorf("%v",string(msg))
		}
		return json.NewDecoder(body).Decode(&res)
	})
	return
}
func GetTransactions(from int,hand func(interface{}) bool )(err error) {

	path := config.Conf.GetAccPath() + "/transactions?"+url.Values{"from":[]string{strconv.Itoa(from)}}.Encode()
	return clientHttp(0,"GET",path,nil,func(statusCode int,body io.Reader )error{
		var res interface{}
		if err =  json.NewDecoder(body).Decode(&res);err != nil {
			return err
		}
		if statusCode != 200 {
			return fmt.Errorf("%v",res)
		}
		for _,p := range res.(map[string]interface{})["pages"].([]interface{}) {
			//fmt.Println(p)
			er := clientHttp(0,"GET",p.(string),nil,func(s int,_body io.Reader)error{
				var _res interface{}
				if er :=  json.NewDecoder(_body).Decode(&_res);er != nil {
					panic(er)
					return er
				}
				if statusCode != 200 {
					return fmt.Errorf("%v",_res)
				}
				for _,db := range _res.(map[string]interface{})["transactions"].([]interface{}) {
					if !hand(db) {
						return io.EOF
					}
				}
				return nil
			})
			if er != nil {
				if er ==  io.EOF {
					break
				}
				log.Println(er)
			}
		}
		return nil

	})

}

func HandleTrades(tp,sl,id string) (*oanda.TradesOrdersRequest,error) {

	//log.Println("update")
	path := config.Conf.GetAccPath() + "/trades/"+id + "/orders"
	req := map[string]interface{}{}
	if tp != "" {
		req["takeProfit"] = &oanda.TakeProfitDetails{Price:tp}
	}
	if sl != "" {
		req["stopLoss"] = &oanda.StopLossDetails{Price:sl}
	}
	da, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	var mr oanda.TradesOrdersRequest
	err = clientHttp(0,"PUT",path,da,func(statusCode int,body io.Reader )error{
		jsondb := json.NewDecoder(body)
		if statusCode != 200 {
			var da interface{}
			err = jsondb.Decode(&da)
			return fmt.Errorf("%d %v %v",statusCode,da,err)
		}
		return jsondb.Decode(&mr)
	})
	//if err != nil {
	//	config.Conf.Log([]byte(err.Error()))
	//	//panic(err)
	//}else{
	//	db,err := json.Marshal(mr)
	//	if err != nil {
	//		panic(err)
	//	}
	//	config.Conf.Log(db)
	//}
	return &mr,err

}
func CloseTrades(tradeId string,longUnits string) (res map[string]interface{}, err error) {

	path := config.Conf.GetAccPath()+"/trades/" + tradeId + "/close"
	//fmt.Println(path)
	//<URL>/v3/accounts/<ACCOUNT>/trades/6397/close"

	val := make(map[string]string)
	//val["longUnits"] = "ALL"
	val["units"] = longUnits
	da, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}
	res = map[string]interface{}{}
	err = clientHttp(0,"PUT",path,da,func(statusCode int,body io.Reader )error{
		err = json.NewDecoder(body).Decode(&res)
		if statusCode != 200 {
			return fmt.Errorf("%d %v %v",statusCode,res,err)
		}
		return err
	})
	//if err != nil {
	//	config.Conf.Log([]byte(err.Error()))
	//	//panic(err)
	//}else{
	//	db,err := json.Marshal(res)
	//	if err != nil {
	//		panic(err)
	//	}
	//	config.Conf.Log(db)
	//}
	return res,err

}

func ClosePosition(InsName string,longUnits string) (*oanda.PositionResponses,error) {

	path := config.Conf.GetAccPath()+"/positions/" + InsName + "/close"
	//fmt.Println(path)
	//log.Println("close")

	val := make(map[string]string)
	//val["longUnits"] = "ALL"
	val["longUnits"] = longUnits
	da, err := json.Marshal(val)
	if err != nil {
		panic(err)
	}
	var mr oanda.PositionResponses
	err = clientHttp(0,"PUT",path,da,func(statusCode int,body io.Reader )error{
		jsondb := json.NewDecoder(body)
		if statusCode != 200 {
			var da interface{}
			err = jsondb.Decode(&da)
			return fmt.Errorf("%d %v %v",statusCode,da,err)
		}
		return jsondb.Decode(&mr)
	})
	//if err != nil {
	//	config.Conf.Log([]byte(err.Error()))
	//	//panic(err)
	//}else{
	//	db,err := json.Marshal(mr)
	//	if err != nil {
	//		panic(err)
	//	}
	//	config.Conf.Log(db)
	//}
	return &mr,err

}
func GetAccSummary() (res map[string]interface{},err error){

	res = map[string]interface{}{}
	err = clientHttp(0,"GET",config.Conf.GetAccPath()+"/summary",nil,func(statusCode int,body io.Reader )error{
		er := json.NewDecoder(body).Decode(&res)
		if er != nil {
			panic(er)
		}
		if statusCode != 200 {
			return fmt.Errorf("%d %v",statusCode,res)
		}
		return nil
	})
	return

}

func HandleOrder(InsName string,unit int, dif , Tp, Sl string) (*oanda.OrderResponse, error) {

	//log.Println("open")
	path := config.Conf.GetAccPath()+"/orders"
	order := oanda.NewMarketOrderRequest(InsName)
	order.SetUnits(unit)
	if dif != "" {
		order.SetTrailingStopLossDetails(dif)
	}
	if Sl != "" {
		order.SetStopLossDetails(Sl)
	}

	if Tp != "" {
		order.SetTakeProfitDetails(Tp)
	}
	//Val["order"] = order

	da, err := json.Marshal(map[string]*oanda.MarketOrderRequest{"order":order})
	if err != nil {
		panic(err)
	}
	//fmt.Println(string(da))
	var mr oanda.OrderResponse
	err = clientHttp(0,"POST",path,da,func(statusCode int,body io.Reader )error{
		jsondb := json.NewDecoder(body)
		if statusCode != 201 {
			var da interface{}
			err = jsondb.Decode(&da)
			if err!= nil {
				panic(err)
			}
			return fmt.Errorf("%d %v",statusCode,da)
		}
		return jsondb.Decode(&mr)

	})
	//if err != nil {
	//	config.Conf.Log([]byte(err.Error()))
	//	//panic(err)
	//}else{
	//	db,err := json.Marshal(mr)
	//	if err != nil {
	//		panic(err)
	//	}
	//	config.Conf.Log(db)
	//}
	return &mr, err

}

type OrderInfo struct {
	ins *oanda.Instrument
	tp float64
	sl float64
	price float64
	timeOut int64
	res *oanda.OrderResponse
	//resUpdate []interface{}
}
func NewOrderInfo(ins *oanda.Instrument) *OrderInfo {
	return &OrderInfo{
		ins:ins,
	}
}
func (self *OrderInfo) TimeOut() int64 {
	return self.timeOut
}
func (self *OrderInfo) String() string {
	return fmt.Sprintf("sl:%s tp:%s resid:%s",self.ins.StandardPrice(self.sl),self.ins.StandardPrice(self.tp),self.GetResId())
}
func (self *OrderInfo) GetRes()*oanda.OrderResponse{
	return self.res
}
//func NewTestOrder(pr *PriceVar,isBuy bool) (*OrderInfo,error) {
//	p := pr.GetLastPr()
//	if p == nil {
//		return nil,fmt.Errorf("last == nil")
//	}
//	fmt.Println(time.Unix(p.Time.Time(),0))
//	or := &OrderInfo{ins:pr.Ins}
//	var _p float64
//	var u int = config.Conf.Units
//	if isBuy {
//		_p = p.Ask()
//		or.sl = _p - pr.Ins.MinimumTrailingStopDistance*2
//		or.tp = _p + pr.Ins.MinimumTrailingStopDistance*2
//	}else{
//		u = -u
//		_p = p.Bid()
//		or.sl = _p + pr.Ins.MinimumTrailingStopDistance*2
//		or.tp = _p - pr.Ins.MinimumTrailingStopDistance*2
//	}
//	return or,or.Post(u)
//
//
//}
func (self *OrderInfo) Close() {
	_,err := ClosePosition(self.ins.Name,"ALL")
	if err == nil {
		//log.Printf("%v\r\n",res)
	}else{
		log.Println(err)
	}
}
func (self *OrderInfo) Update() {
	_,err :=HandleTrades(self.ins.StandardPrice(self.tp),self.ins.StandardPrice(self.sl),self.GetResId())
	if err == nil {
		//log.Printf("%v\r\n",res)
	}else{
		log.Println(err)
	}
	return
}
func (self *OrderInfo) UpdateNew(tp,sl,pr float64,timeOut int64) {
	self.sl = sl
	self.tp = tp
	self.price = pr
	self.timeOut = timeOut
	self.PostNew()
}
func (self *OrderInfo) PostNew() {

	unit := self.getUnit(config.Conf.Units)
	if unit == 0 {
		return
	}
	if  self.tp < self.sl {
		unit = -unit
	}
	//diff := self.ins.StandardPrice(math.Abs(self.sl - self.price))
	self.res,_ = HandleOrder(self.ins.Name,unit,"",self.ins.StandardPrice(self.tp),self.ins.StandardPrice(self.sl))

}
func (self *OrderInfo) Check(p float64,dateTime int64) {

	oid := self.GetResId()
	if oid == "" {
		return
	}
	defer func(){
		if self.timeOut != 0 {
			if self.GetResId() == "" {
				self.timeOut = 0
			}
		}
	}()
	if CheckTrades(oid) {
		self.res = nil
		return
	}

	//if ((dateTime - self.GetResTime()) > self.timeOut) ||
	if (p>self.sl && p>self.tp) ||
		(p< self.sl && p<self.tp) {
		_,err := CloseTrades(oid,"ALL")
		if err != nil {
			return
		}
		if CheckTrades(oid) {
			self.res = nil
			return
		}
		_,err = ClosePosition(self.ins.Name,"ALL")
		if err != nil {
			return
		}
		if CheckTrades(oid) {
			self.res = nil
			return
		}
	}

}

func (self *OrderInfo) getUnit(unit int) int {
	return int((MarginRate*float64(unit))/((self.sl+self.tp)/2))
}
func (self *OrderInfo) Post(unit int) (err error) {

	self.res,err = HandleOrder(self.ins.Name,self.getUnit(unit),"",self.ins.StandardPrice(self.tp),self.ins.StandardPrice(self.sl))
	//if err != nil {
	//	fmt.Println(self.res)
	//}
	return err

}
func (self *OrderInfo) GetResPrice() float64 {
	p,err := strconv.ParseFloat(string(self.res.OrderFillTransaction.Price),64)
	if err != nil {
		panic(err)
	}
	return p
}
func (self *OrderInfo) GetResTime() int64 {
	if self.res == nil {
		return 0
	}
	return self.res.OrderFillTransaction.Time.Time()
}
func (self *OrderInfo) GetResId() string {

	if self.res == nil {
		return ""
	}
	return string(self.res.OrderFillTransaction.Id)

}

