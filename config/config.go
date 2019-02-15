package config
import(
	"flag"
	"github.com/BurntSushi/toml"
	"os"
	"log"
	"path/filepath"
	"time"
	//"net/http"
)
const (
	Authorization string = "d35019efd0aa18b844697b3f9cce744f-a57d33002bb3b74297cb2e216d348b58"
	Host string = "https://api-fxpractice.oanda.com/v3"
	StreamHost string = "https://Stream-fxpractice.oanda.com/v3"
	TimeFormat = "2006-01-02T15:04:05"
	DBType string = "sqlite3"
)
var (
	FileName   = flag.String("c", "conf.log", "config log")
	InsName   = flag.String("n", "", "Sample EUR_JPY|EUR_USD")
	Conf *Config
	Granularity = []*Gran{
			&Gran{"S5",5},
			&Gran{"S10" , 10},
			&Gran{"S15" , 15},
			&Gran{"S30" , 30},
			&Gran{"M1"  , 60},
			&Gran{"M2"  , 60 * 2},
			&Gran{"M4"  , 60 * 4},
			&Gran{"M5"  , 60 * 5},
			&Gran{"M10" , 60 * 10},
			&Gran{"M15" , 60 * 15},
			&Gran{"M30" , 60 * 30},
			&Gran{"H1"  , 3600},
			&Gran{"H2"  , 3600 * 2},
			&Gran{"H3"  , 3600 * 3},
			&Gran{"H4"  , 3600 * 4},
			&Gran{"H6"  , 3600 * 6},
			&Gran{"H8"  , 3600 * 8},
			&Gran{"H12" , 3600 * 12},
			&Gran{"D"   , 3600 * 24},
			&Gran{"w"   , 3600 * 24*7},
		}
)
func GetTime() time.Time {
	loc,err := time.LoadLocation("Etc/GMT-3")
	if err != nil {
		panic(err)
	}
	return time.Now().In(loc)
}
type Gran struct {
	name string
	val int64
}
func (self *Gran) Name()string {
	return self.name
}
func (self *Gran) Val()int64 {
	return self.val
}

type Config struct {

	AccountID string
	Proxy string
	Templates string
	//BEGINTIME string
	Port string
	//InsName string
	Server bool
	//Price bool
	Units int
	//Granularity []*Gran
	LogPath string
	//DbPath string
	KvDbPath string
	Val float64

}
func GetGran(t int64) *Gran{

	var lastgr *Gran = nil
	for _,gr := range Granularity{
		if gr.val > t {
			if lastgr == nil {
				return nil
			}else{
				if (gr.val - t) > (t - lastgr.val){
					return lastgr
				}else{
					return gr
				}
			}
		}else{
			lastgr = gr
		}
	}
	return nil

}
func (self *Config) GetStreamAccPath() string {
	return StreamHost+"/accounts/"+self.AccountID
}
func (self *Config) GetAccPath() string {
	return Host+"/accounts/"+self.AccountID
}

func (self *Config) Log(db []byte) {
	if _,err := os.Stat(self.LogPath); err != nil {
		err = os.Mkdir(self.LogPath,0600)
		if err != nil {
			panic(err)
		}
	}
	f,err := os.OpenFile(
		filepath.Join(
			self.LogPath,
			GetTime().Format("20060102.log")),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_SYNC,
		0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _,err = f.Write(append(db,'\n')); err != nil {
		panic(err)
	}

}
func (self *Config) Save() {

	fi,err := os.OpenFile(*FileName,os.O_CREATE|os.O_WRONLY,0600)
	//fi,err := os.Open(FileName)
	if err != nil {
		log.Fatal(err)
	}
	defer fi.Close()
	e := toml.NewEncoder(fi)
	err = e.Encode(self)
	if err != nil {
		log.Fatal(err)
	}

}

func NewConfig()  *Config {

	var c Config
	_,err := os.Stat(*FileName)
	if err != nil {
		c.AccountID = "101-011-2471429-001"
		c.Port=":8080"
		c.Templates = "./templates/*"
		c.Units = 100
		c.Server = true
		c.LogPath = "TrLog"
		//c.DbPath = "db.db"
		c.KvDbPath = "kvdb.db"
		c.Val = 3
		//c.LoadGranularity()
		c.Save()
	}else{
		if _,err := toml.DecodeFile(*FileName,&c);err != nil {
			log.Fatal(err)
		}
	}
	//if *InsName != "" {
	//	c.InsName = *InsName
	//}
	return &c

}

func init(){
	flag.Parse()
	Conf = NewConfig()
}
