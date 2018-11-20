package main
import(
	"fmt"
	"github.com/zaddone/operate/request"
	_ "github.com/zaddone/operate/server"
	"strings"
)
func main(){
	var cmd string
	for {
		fmt.Scanf("%s",&cmd)
		switch strings.ToLower(cmd) {
		case "show":
			request.Show()
	//	case "time":
	//		request.ShowTime()
		}
		fmt.Println(cmd)
		cmd = ""
	}
}
