package server
import(

	"fmt"
	"time"
	"net/http"
	"github.com/zaddone/operate/config"
	"github.com/gin-gonic/gin"
	//"github.com/zaddone/operate/oanda"
	"github.com/zaddone/operate/request"
	"encoding/json"
	"strings"
	"io/ioutil"
	"io"

)
var (
	Router *gin.Engine
)
func init(){

	if !config.Conf.Server {
		return
	}
	Router = gin.Default()
	Router.LoadHTMLGlob(config.Conf.Templates)

	Router.GET("/",func(c *gin.Context){
		c.HTML(http.StatusOK,"index.tmpl",nil)
	})

	Router.GET("/open",func(c *gin.Context){
		var res interface{}
		err := request.ClientHttp(0,"GET",
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
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
		}
		c.JSON(http.StatusOK,res)
	})
	Router.GET("/test",func(c *gin.Context){
		var orderFill []interface{}
		t,err := time.Parse(config.TimeFormat,"2018-09-01T00:00:00")
		if err != nil {
			panic(err)
		}
		err = request.GetTransactions(int(t.Unix()),func(db interface{}) bool {
			dbm := db.(map[string]interface{})
			if dbm["type"] != "ORDER_FILL" {
				return true
			}
			//fmt.Println(dbm)
			orderFill = append(orderFill,dbm)
			if len(orderFill) >100 {
				return false
			}
			return true
		})
		if err != nil {
			c.JSON(http.StatusNotFound,err)
			return
		}
		c.JSON(http.StatusOK,orderFill)
	})
	Router.GET("/order/:isBuy/:insName",func(c *gin.Context){

		v,ok := request.InsSet.Load(strings.ToUpper(c.Param("insName")))
		if !ok {
			c.JSON(http.StatusNotFound,gin.H{"msg":fmt.Sprintf("Fount not insName %s",c.Param("insName"))})
			return
		}
		or,err := request.NewTestOrder(v.(*request.PriceVar),strings.ToLower(c.Param("isBuy"))=="buy")
		if err != nil {
			c.JSON(http.StatusNotFound,gin.H{"msg":err.Error()})
			return
		}
		c.JSON(http.StatusOK,gin.H{"id":or.GetResId(),"date":time.Unix(or.GetResTime(),0)})

	})
	Router.Run(config.Conf.Port)

}
