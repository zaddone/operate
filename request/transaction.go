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

//func TransactionsRange(from int,page int,hand func(interface{})error) (to int,err error) {
//	err = clientHttp(0,"GET",
//	config.Conf.GetAccPath() + "/transactions/idrange?"+url.Values{"from":[]string{strconv.Itoa(from)},"to":[]string{strconv.Itoa(from+page)}}.Encode(),nil,func(sta int,body io.Reader )(er error) {
//		if sta != 200 {
//			msg,_ := ioutil.ReadAll(body)
//			return fmt.Errorf("%d %s",sta,string(msg))
//		}
//		var _res interface{}
//		if er =  json.NewDecoder(body).Decode(&_res) ;er != nil {
//			return er
//		}
//		res :=_res.(map[string]interface{})
//		to,er =strconv.Atoi(res["lastTransactionID"].(string))
//		if er != nil {
//			return er
//		}
//		for _,_tr := range res["transactions"].([]interface{}) {
//			if er = hand(_tr);er != nil{
//				return er
//			}
//		}
//		return nil
//	})
//	return
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
		//fmt.Println(trm)
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
					return fmt.Errorf("%d %s",sta,string(msg))
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
			log.Println("transactions/stream",err)
		}
	}

}
