package main
import(
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/zaddone/operate/request"
	"github.com/zaddone/operate/oanda"
	"github.com/zaddone/operate/cache"
	"github.com/zaddone/operate/config"
	"github.com/zaddone/operate/server"
	"strings"
	"sync"
	"io/ioutil"
	"bufio"
	"encoding/json"
	"net/url"
	"io"
	"log"
)
var (
	InsSet sync.Map = sync.Map{}
)
func main(){
	syncPrice()
	server.Router.Run(config.Conf.Port)
	//var cmd string
	//for {
	//	fmt.Scanf("%s",&cmd)
	//	switch strings.ToLower(cmd) {
	//	case "show":
	//		//request.Show()
	////	case "time":
	////		request.ShowTime()
	//	}
	//	fmt.Println(cmd)
	//	cmd = ""
	//}
}
func ShowInsSet() (m map[string]interface{}){
	m = map[string]interface{}{}
	InsSet.Range(func(k,v interface{})bool{
		pv :=v.(*cache.Cache)
		m[k.(string)] = map[string]interface{}{"ins":pv.Ins,"endTime":pv.EndTime()}
		return true
	})
	return
}
func syncPrice() {
	//ins_url := url.Values{}
	var Ins []string
	var b *bolt.Bucket
	err := config.HandDB(func(db *bolt.DB)error{
		return db.View(func(tx *bolt.Tx) error {
			b = tx.Bucket(request.Ins_key)
			if b == nil {
				return nil
			}
			return b.ForEach(func(k,v []byte)error{
				_ins := &oanda.Instrument{}
				err := json.Unmarshal(v,_ins)
				if err != nil {
					panic(err)
				}
				InsSet.Store(string(k),cache.NewCache(_ins))
				Ins = append(Ins,string(k))
				return nil
			})
		})
	})
	if err != nil {
		panic(err)
	}
	if b == nil {
		err = request.DownAccountProperties()
		if err != nil {
			panic(err)
		}
		syncPrice()
	}else{
		go syncGetPriceVar(&url.Values{"instruments":[]string{strings.Join(Ins,",")}})
	}

}
func syncGetPriceVar(ins_url *url.Values){

	var err error
	var lr,r []byte
	var p bool
	for{
		err = request.ClientHttp(0,
		"GET",
		config.Conf.GetStreamAccPath()+"/pricing/stream?"+ins_url.Encode(),
		nil,
		func(statusCode int,data io.Reader) error {
			if statusCode != 200 {
				msg,_ := ioutil.ReadAll(data)
				return fmt.Errorf("%s",string(msg))
			}
			buf := bufio.NewReader(data)
			for{
				r,p,err = buf.ReadLine()
				//fmt.Println(string(r),p)
				if p {
					//fmt.Println(string(r))
					lr = r
				}else if len(r)>0 {
					if lr != nil {
						r = append(lr,r...)
						lr = nil
					}

					var d oanda.Price
					er := json.Unmarshal(r,&d)
					if er != nil {
						//log.Println(er,string(r))
						continue
					}
					//fmt.Printf("%s\r",request.GetNowTime(d.Time.Time()))
					name := string(d.Instrument)

					if request.GetEndDaySec(d.Time.Time())<60*10 {
						er = request.CloseAllTrades()
						if er != nil {
							log.Println(er)
						}
						continue
					}
					if name != "" {
						v,ok := InsSet.Load(name)
						if !ok {
							panic(name)
						}
						go v.(*cache.Cache).AddPrice(oanda.NewEasyPrice(&d))
					}

				}
				if err != nil {
					if err != io.EOF {
						//panic(err)
						//log.Println("line",err)
					}
					return err
				}
			}
			return nil
		})
		if err != nil {
			//panic(err)
			//log.Println("/pricing/stream",err)
		}
	}

}

