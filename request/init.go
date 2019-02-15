package request
import(
	"net/http"
	"github.com/boltdb/bolt"
	"github.com/zaddone/operate/config"
	//"github.com/zaddone/operate/oanda"
	//"github.com/zaddone/operate/cache"
	"io"
	"net/url"
	"strings"
	"time"
	"compress/gzip"
	"fmt"
	"encoding/json"
	//"log"
	//"sync"
	//"bufio"
	"bytes"
	"strconv"
	//"math"
)
var (
	Header  http.Header
	Ins_key []byte = []byte("instruments")
	AccountSummary map[string]interface{}
	MarginRate float64
)
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
			//fmt.Printf("%v\r\n",body)
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
		//fmt.Println(err)
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

func DownAccountProperties() error {
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
			//fmt.Println(err)
			break
		}
	}
	return nil

}

