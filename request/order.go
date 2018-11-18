package request
import(
	//"io/ioutil"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/oanda"
	//"log"
	"encoding/json"
	"net/url"
	"io"
	"fmt"
	"time"
	"strconv"
	"log"
	//"math"
)
//func GetOpenTrades(){
//	err := clientHttp(0,"GET",
//	config.Conf.GetAccPath() + "/openTrades",nil,
//	func(statusCode int,body io.Reader) (er error){
//		if statuscode != 200 {
//			msg,_ := ioutil.ReadAll(body)
//			return fmt.errorf("%v",string(msg))
//		}
//		var _res interface{}
//		if er =  json.NewDecoder(body).Decode(&_res) ;er != nil {
//			return er
//		}
//	})
//}
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
	path := config.Conf.GetAccPath() + "/trades/"+id + "/orders"
	da, err := json.Marshal(map[string]interface{}{"takeProfit":&oanda.TakeProfitDetails{Price:tp},"stopLoss":&oanda.StopLossDetails{Price:sl}})
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
	return &mr,err

}

func ClosePosition(InsName string,longUnits string) (*oanda.PositionResponses,error) {

	path := config.Conf.GetAccPath()+"/positions/" + InsName + "/close"

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
	return &mr,err

}

func HandleOrder(InsName string,unit int, dif , Tp, Sl string) (*oanda.OrderResponse, error) {

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
	//	panic(err)
	//}
	//fmt.Println(mr,err)
	return &mr, err

}


type OrderInfo struct {
	ins *Instrument
	tp float64
	sl float64
	res *oanda.OrderResponse
	//resUpdate []interface{}
}
func (self *OrderInfo) String() string {
	return fmt.Sprintf("sl:%s tp:%s resid:%s",self.ins.StandardPrice(self.sl),self.ins.StandardPrice(self.tp),self.GetResId())
}
func (self *OrderInfo) GetRes()*oanda.OrderResponse{
	return self.res
}
func NewTestOrder(pr *PriceVar,isBuy bool) (*OrderInfo,error) {
	p := pr.GetLastPr()
	if p == nil {
		return nil,fmt.Errorf("last == nil")
	}
	fmt.Println(time.Unix(p.Time.Time(),0))
	or := &OrderInfo{ins:pr.Ins}
	var _p float64
	var u int = config.Conf.Units
	if isBuy {
		_p = p.Ask()
		or.sl = _p - pr.Ins.MinimumTrailingStopDistance*2
		or.tp = _p + pr.Ins.MinimumTrailingStopDistance*2
	}else{
		u = -u
		_p = p.Bid()
		or.sl = _p + pr.Ins.MinimumTrailingStopDistance*2
		or.tp = _p - pr.Ins.MinimumTrailingStopDistance*2
	}
	return or,or.Post(u)


}
func (self *OrderInfo) Close() {
	res,err := ClosePosition(self.ins.Name,"ALL")
	if err == nil {
		log.Printf("%v\r\n",res)
	}else{
		log.Println(err)
	}
}
func (self *OrderInfo) Update() {
	res,err :=HandleTrades(self.ins.StandardPrice(self.tp),self.ins.StandardPrice(self.sl),self.GetResId())
	if err == nil {
		log.Printf("%v\r\n",res)
	}else{
		log.Println(err)
	}
	return
}
func (self *OrderInfo) PostNew() {

	unit := config.Conf.Units
	if  (self.tp - self.sl)<0 {
		unit = -unit
	}
	err := self.Post(unit)
	if err == nil {
		log.Printf("%v\r\n",self.res)
	}else{
		log.Println(err)
	}

}
func (self *OrderInfo) Post(unit int) (err error) {

	self.res,err = HandleOrder(self.ins.Name,unit,"",self.ins.StandardPrice(self.tp),self.ins.StandardPrice(self.sl))
	//if err != nil {
	//	fmt.Println(self.res)
	//}
	return err

}
func (self *OrderInfo) GetResTime() int64 {
	return self.res.OrderFillTransaction.Time.Time()
}
func (self *OrderInfo) GetResId() string {

	if self.res == nil {
		return ""
	}
	return string(self.res.OrderFillTransaction.Id)

}

