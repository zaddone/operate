package main
import(
	"net/http"
	"net/url"
	"fmt"
	"strings"
	"io"
	"compress/gzip"
	"time"
	"bytes"
	"bufio"
)
var (
	Header  http.Header
)
func main(){
	Header = http.Header{}
	//Header.Set("Authorization", "Bearer "+ config.Authorization)
	Header.Set("Connection", "Keep-Alive")
	Header.Set("Accept-Datetime-Format", "UNIX")
	Header.Set("Content-type", "application/json")
	err := clientHttp(0,"GET","https://www.baidu.com",nil,func(statusCode int,data io.Reader)error {
		buf := bufio.NewReader(data)
		for{
			li,pr,err := buf.ReadLine()
			if !pr{
				fmt.Println(string(li))
			}
			if err != nil {
				return err
			}
		}
		return nil

	})
	if err != nil {
		if err != io.EOF {
			panic(err)
		}
	}
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
