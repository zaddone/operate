package request
import(
	//"reflect"
	"net/url"
	"github.com/boltdb/bolt"
	"encoding/json"
	"io"
	"log"
	"fmt"
	"bufio"
	"io/ioutil"
	"github.com/zaddone/operate/config"
	//"github.com/zaddone/operate/oanda"
	"strconv"
)
//func sliceToStruct(src []interface{},obj interface) {
//
//	t := reflect.TypeOf(obj)
//	if t.Kill() != reflect.Slice {
//		return fmt.Errorf("%v",t.Kill())
//	}
//	v := reflect.ValueOf(obj)
//
//
//	for i,_v := range src {
//		switch _v.(type) {
//		case map[string]interface{}:
//			//var o interface{}
//		case interface{}:
//			reflect.Append(v,_v)
//		}
//	}
//
//}
//func mapToStruct(src map[string]interface{},obj interface{}){
//
//	v := reflect.ValueOf(obj)
//	for _k,_v := range src {
//		switch _v.(type){
//		case []interface{}:
//			sl := v.FieldByName(_k)
//			ddsl.Kind() == reflect.Slice
//			sliceToStruct(_v.([]interface{}),v.FieldByName(_k).Interface())
//		case map[string]interface{}:
//			mapToStruct(_v.(map[string]interface{}),v.FieldByName(_k).Interface())
//		case interface{}:
//			v.FieldByName(_k).Set(reflect.ValueOf(_v))
//		}
//	}
//
//}

//func NewOrderFillTransaction(d map[string]interface{}) (*OrderFillTransaction) {
//
//	var tr OrderFillTransaction
//	v := reflect.ValueOf(&tr)
//	for k,v := range 
//
//
//}
//func newTransaction (trType string) (transaction interface{}) {
//
//	switch trType {
//		case "HEARTBEAT" :
//			return new(oanda.TransactionHeartBeat)
//		case "ORDER_FILL":
//			return new(oanda.OrderFillTransaction)
//		case "ORDER_CANCEL":
//			return new(oanda.OrderCancelTransaction)
//		case "MARKET_ORDER":
//			return new(oanda.MarketOrderTransaction)
//		case "DAILY_FINANCING":
//			return new(oanda.DailyFinancingTransaction)
//		case "TAKE_PROFIT_ORDER":
//			return new(oanda.TakeProfitOrderTransaction)
//		case "STOP_LOSS_ORDER":
//			return new(oanda.StopLossOrderTransaction)
//		case "TRAILING_STOP_LOSS_ORDER":
//			return new(oanda.TrailingStopLossOrderTransaction)
//		case "MARKET_ORDER_REJECT":
//			return new(oanda.MarketOrderRejectTransaction)
//		case "CREATE":
//			return new(oanda.CreateTransaction)
//		case "CLIENT_CONFIGURE":
//			return new(oanda.ClientConfigureTransaction)
//		case "TRANSFER_FUNDS":
//			return new(oanda.TransferFundsTransaction)
//		case "ORDER_CANCEL_REJECT":
//			return new(oanda.OrderCancelRejectTransaction)
//		default:
//			panic(string(trType))
//	}
//
//}
//
//func handTransaction(db []byte) (err error){
//
//	var d map[string]interface{}
//	err = json.Unmarshal(db,&d)
//	if err != nil {
//		return err
//	}
//	tr := newTransaction(d["type"].(string))
//	return json.Unmarshal(db,tr)
//
//
//}
func runTransactionsStream(){
	syncTransaction(func(tr interface{}){
		if db,err:=json.Marshal(tr) ; err == nil{
			config.Conf.Log(db)
		}
		trm := tr.(map[string]interface{})
		if trm["type"] != "ORDER_FILL" {
			return
		}
		fmt.Println(trm)
		if trm["tradesClosed"] == nil {
			return
		}
		if v,ok := InsSet.Load(trm["instrument"].(string));ok{
			pv :=v.(*PriceVar)
			if pv.order == nil {
				return
			}
			orderId :=pv.order.GetResId()
			if  orderId == ""{
				return
			}
			for _,info := range trm["tradesClosed"].([]interface{}) {
				_info :=info.(map[string]interface{})
				closeid := _info["tradeID"].(string)
				if orderId != closeid {
					continue
				}
				pv.order.res = nil
				pv.CountOrder++
				if pl,err := strconv.ParseFloat(_info["realizedPL"].(string),64);err == nil {
					pv.Pl += pl
				}else{
					panic(err)
				}
			}
		}

	})

}
func syncTransaction(hand func(interface{})){
	var lastid int
	err := config.ViewKvDB([]byte("tran"),func(b *bolt.Bucket)error{
		if v,er :=strconv.Atoi(string( b.Get([]byte("lastid")))); er == nil {
			lastid = v
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
	TransactionStream(func(db []byte)error{
		var tr map[string]interface{}
		if er :=  json.Unmarshal(db,&tr) ; er != nil {
			return er
		}
		if tr["id"] == nil {
			return nil
		}
		_id :=tr["id"].(string)
		id,er := strconv.Atoi(_id)
		if er != nil {
			return er
		}
		if (lastid != 0) && (lastid+1 != id) {
			er = clientHttp(0,"GET",
			config.Conf.GetAccPath() + "/transactions/idrange?"+url.Values{"from":[]string{strconv.Itoa(lastid+1)},"to":[]string{strconv.Itoa(id-1)}}.Encode(),nil,func(sta int,body io.Reader ) error {
				if sta != 200 {
					msg,_ := ioutil.ReadAll(body)
					return fmt.Errorf("%d %s",sta,msg)
				}
				var _res interface{}
				if er =  json.NewDecoder(body).Decode(&_res) ;er != nil {
					return er
				}
				for _,_tr := range _res.(map[string]interface{})["transactions"].([]interface{}) {
					hand(_tr)
				}
				return nil
			})
		}
		lastid = id
		er = config.UpdateKvDB([]byte("tran"),func(b *bolt.Bucket)error{
			b.Put([]byte("lastid"),[]byte(_id))
			return nil
		})
		if er != nil {
			panic(er)
		}
		hand(tr)
		return nil

	})
}

func TransactionStream(hand func([]byte)error) (err error){
	//var err error
	var lr []byte
	var r []byte
	var p bool
	for{
		err = clientHttp(0,
		"GET",
		config.Conf.GetStreamAccPath()+"/transactions/stream",
		nil,
		func(statusCode int,data io.Reader) error {
			if statusCode != 200 {
				msg,_ := ioutil.ReadAll(data)
				return fmt.Errorf("%d %s",statusCode,msg)
			}
			buf := bufio.NewReader(data)
			for{
				r,p,err = buf.ReadLine()
				//fmt.Println(string(r),p)
				if p {
					fmt.Println(string(r))
					lr = r
				}else if len(r)>0 {
					if lr != nil {
						r = append(lr,r...)
						lr = nil
					}
					if er := hand(r);er != nil{
						log.Println(er)
					}
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
			log.Println(err)
		}
	}

}
