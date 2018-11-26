package request
import(
	"net/http"
	"github.com/boltdb/bolt"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/oanda"
	"io"
	"net/url"
	"strings"
	"time"
	"compress/gzip"
	"fmt"
	"encoding/json"
	"log"
	"sync"
	"bufio"
	"bytes"
	"strconv"
	//"math"
)
var (
	Header  http.Header
	Ins_key []byte = []byte("instruments")
	//InsMap map[string]*Instrument = map[string]*Instrument{}
	InsSet sync.Map = sync.Map{}

	AccountSummary map[string]interface{}
	MarginRate float64
	//isEnd bool = false
	//ServerTime int64
)
func ShowInsSet() (m map[string]interface{}){
	m = map[string]interface{}{}
	InsSet.Range(func(k,v interface{})bool{
		pv :=v.(*PriceVar)
		pr := pv.GetLastPr()
		if pr == nil {
			return true
		}
		m[k.(string)] = map[string]interface{}{"ins":pv,"price":pr}
		return true
	})
	return
}
func GetNowTime(timeu int64) time.Time {
	loc,err := time.LoadLocation("Etc/GMT-3")
	if err != nil {
		panic(err)
	}
	return time.Unix(timeu,0).In(loc)
}
func GetEndDaySec(timeu int64) int64 {
	now := GetNowTime(timeu)
	end := time.Date(now.Year(),now.Month(),now.Day(),0,0,0,0,now.Location()).AddDate(0,0,1)
	return end.Unix() - timeu
}
//func ShowTime(){
//	now := time.Now()
//	fmt.Println(GetNowTime(),now,float64(GetEndDaySec())/3600)
//}
func Show(){
	//num :=0
	InsSet.Range(func(k,v interface{})bool{
		p :=v.(*PriceVar)
		if p.last != nil {
			fmt.Println(k,p.last.PriceInterval(),p.PriceInterval(),p.TimeInterval())
		}
		//_v :=v.(*PriceVar)
		//val := _v.get()
		//p := math.Pow(10,_v.Ins.DisplayPrecision)
		//bv := _v.Ins.MinimumTrailingStopDistance/val
		//if bv >2{
		//	fmt.Println(k,bv,_v.Ins.MinimumTrailingStopDistance*p,val*p)
		//	num++
		//}
		return true
	})
}

func init(){


	Header = http.Header{}
	Header.Set("Authorization", "Bearer "+ config.Authorization)
	Header.Set("Connection", "Keep-Alive")
	Header.Set("Accept-Datetime-Format", "UNIX")
	Header.Set("Content-type", "application/json")


	var err error
	AccountSummary,err = GetAccSummary()
	if err != nil {
		panic(err)
	}
	MarginRate,err = strconv.ParseFloat(AccountSummary["account"].(map[string]interface{})["marginRate"].(string),64)
	if err != nil {
		panic(err)
	}
	MarginRate = 1/MarginRate
	syncPrice()

}

func syncGetPriceVar(ins_url *url.Values){

	var err error
	var lr,r []byte
	var p bool
	//Now := time.Now().Unix()
	//var count int64
	//fmt.Println(ins_url.Encode())
	for{
		err = clientHttp(0,
		"GET",
		config.Conf.GetStreamAccPath()+"/pricing/stream?"+ins_url.Encode(),
		nil,
		func(statusCode int,data io.Reader) error {
			if statusCode != 200 {
				msg:=""
				var by [1024]byte
				var n int
				for{
					n,err = data.Read(by[0:])
					msg+= string(by[:n])
					if err != nil {
						break
					}
				}
				return fmt.Errorf("%d %v %v %v",statusCode,msg,err)
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

					var d oanda.Price
					er := json.Unmarshal(r,&d)
					if er != nil {
						log.Println(er,string(r))
						continue
					}
					name := string(d.Instrument)

					if GetEndDaySec(d.Time.Time())<60*10 {
						er = CloseAllTrades()
						if er != nil {
							log.Println(er)
						}
						continue
					}
					if name != "" {
						//ServerTime = d.Time.Time()
						//count++
						//timeDif:=(ServerTime - Now)
						//fmt.Printf("%s %d %d\r",time.Unix(ServerTime,0),timeDif,count)
						//fmt.Println(d)
						v,ok := InsSet.Load(name)
						if !ok {
							panic(name)
						}
						go v.(*PriceVar).AddPrice(&d)
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
			//panic(err)
			log.Println("/pricing/stream",err)
		}
	}

}

func syncPrice() {

	//ins_url := url.Values{}
	var Ins []string
	var b *bolt.Bucket
	err := config.HandDB(func(db *bolt.DB)error{
		return db.View(func(tx *bolt.Tx) error {
			b = tx.Bucket(Ins_key)
			if b == nil {
				return nil
			}
			return b.ForEach(func(k,v []byte)error{
				_ins := &Instrument{}
				err := json.Unmarshal(v,_ins)
				if err != nil {
					panic(err)
				}
				InsSet.Store(string(k),NewPriceVar(_ins))
				Ins = append(Ins,string(k))
				return nil
			})
		})
	})
	if err != nil {
		panic(err)
	}
	if b == nil {
		err = downAccountProperties()
		if err != nil {
			panic(err)
		}
		syncPrice()
	}else{
		//le := len(Ins)
		//var I int
		//for i:=0;i<le;{
		//	I=i+50
		//	go syncGetPriceVar(&url.Values{"instruments":[]string{strings.Join(Ins[i:I],",")}})
		//	i = i
		//}
		go syncGetPriceVar(&url.Values{"instruments":[]string{strings.Join(Ins,",")}})
	}

}

func ClientHttp(num int ,methob string,path string,body interface{}, hand func(statusCode int,data io.Reader )error) error {
	return clientHttp(num,methob,path,body,hand)
}

func clientHttp(num int ,methob string,path string,body interface{}, hand func(statusCode int,data io.Reader )error) error {
//func clientHttp(num int ,methob string,path string,body *url.Values, hand func(statusCode int,data io.Reader )error) error {
	if num >5 {
		return fmt.Errorf("num >5")
	}

	var r io.Reader
	if body != nil {
		switch body.(type){
		case url.Values:
			r = strings.NewReader(body.(url.Values).Encode())
		case []byte:
			r = bytes.NewReader(body.([]byte))
		case string:
			r = strings.NewReader(body.(string))
		default:
			fmt.Printf("%v\r\n",body)
			panic(0)

		}
		//fmt.Println(body.Encode())
	}
	Req, err := http.NewRequest(methob, path, r)
	//fmt.Println(Req.Form)
	if err != nil {
		return err
	}
	Req.Header = Header
	cli := http.Client{}
	res, err := cli.Do(Req)
	if err != nil {
		fmt.Println(err)
		time.Sleep(time.Second*5)
		return clientHttp(num+1,methob,path,body,hand)
	}

	var reader io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, _ = gzip.NewReader(res.Body)
		//defer reader.Close()
	default:
		reader = res.Body
	}
	err = hand(res.StatusCode,reader)
	reader.Close()
	return err

}
func downAccountProperties() error {
	//fmt.Println("down")
	return clientHttp(0,"GET",config.Conf.GetAccPath()+"/instruments",nil,func(statusCode int,data io.Reader)(err error){
		jsondb := json.NewDecoder(data)
		if statusCode != 200 {
			var da interface{}
			err = jsondb.Decode(&da)
			return fmt.Errorf("%d %v %v",statusCode,da,err)
		}
		type ResInstrument struct {
			Instruments []map[string]interface{} `json:"instruments"`
		}
		da := &ResInstrument{}
		err = jsondb.Decode(da)
		if err != nil {
			return err
		}
		return config.UpdateKvDB([]byte("instruments"),func(b *bolt.Bucket)error{
			for _,_ins := range da.Instruments {
				var ins Instrument
				err = ins.load(_ins)
				if err != nil {
					return err
				}
				db,err :=  json.Marshal(ins)
				if err != nil {
					return err
				}
				err = b.Put([]byte(ins.Name),db)
				if err != nil {
					return err
				}
			}
			return nil
		})

	})
}

func candlesHandle(path string, f func(c interface{}) error) (err error) {
	da := make(map[string]interface{})
	err = clientHttp(0,"GET",path,nil,func(statusCode int,data io.Reader)(err error){
		jsondb := json.NewDecoder(data)
		if statusCode != 200 {
			var errda interface{}
			err = jsondb.Decode(&errda)
			return fmt.Errorf("%d %v %v",statusCode,errda,err)
		}
		return jsondb.Decode(&da)
	})
	if err != nil {
		return err
	}
	ca := da["candles"].([]interface{})
	lc := len(ca)
	if lc == 0 {
		return fmt.Errorf("candles len = 0")
	}
	for _, c := range ca {
		err = f(c)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
	return nil

}

