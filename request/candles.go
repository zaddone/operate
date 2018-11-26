package request
import(
	"github.com/zaddone/operate/config"
	"strconv"
	"fmt"
	"math"
	"net/url"
)
type node struct {
	ca *Candles
	dis float64
	n int
}
type CanCache struct {
	nodes []*node
	cans []*Candles
	endMax float64

}
func (self *CanCache) GetDis() float64 {
	if self.endMax!=0 {
		return self.endMax
	}
	le := len(self.nodes)
	if le == 0 {
		return 0
	}
	return self.nodes[le-1].dis
}

func NewCanCache(insName string,gr *config.Gran,Count int) (ca *CanCache,err error) {
	if Count>100 {
		Count = 100
	}
	ca = &CanCache{}
	ca.cans = make([]*Candles,0,Count)
	var can *Candles
	var diff,sum,dis float64
	var num ,begin int
	err = candlesHandle(
		fmt.Sprintf(
			"%s/instruments/%s/candles?%s",
			config.Host,
			insName,
			url.Values{
				"granularity": []string{gr.Name()},
				"price": []string{"M"},
				"count": []string{fmt.Sprintf("%d", Count)},
				//"dailyAlignment":[]string{"3"},
			}.Encode(),
		),
		func(c interface{})error{
			can,err = NewCandles(c.(map[string]interface{}))
			if err != nil {
				panic(err)
			}
			num = 0
			ca.endMax = 0
			for i,_can := range ca.cans[begin:] {
				sum += _can.getMidLong()
				diff = can.getVal() - _can.getVal()
				if (dis>0) == (diff>0) {
					continue
				}
				if math.Abs(diff) > math.Abs(ca.endMax) {
					ca.endMax = diff
					num = i
				}
			}
			if (num != 0) &&
			(math.Abs(ca.endMax) > (sum/float64(Count))) {
				begin += num
				ca.nodes = append(ca.nodes,&node{ca.cans[begin],ca.endMax,begin})
				dis = ca.endMax
			}
			ca.cans = append(ca.cans,can)
			return nil
		},
	)
	//if ca.endMax == 0 {
	//	le := len(ca.nodes)
	//	if le >0 {
	//		ca.endMax = ca.nodes[le-1].dis
	//	}
	//}
	if err != nil {
		return
	}
	return

}
type Candles struct {
	Mid    [4]float64
	Time   int64
	Volume float64
	Val    float64
	Scale  int64
}
func (self *Candles) getVal() float64 {
	if self.Val == 0 {
		var sum float64 = 0
		for _, m := range self.Mid {
			sum += m
		}
		self.Val = sum / 4
	}
	return self.Val
}
func (self *Candles) getMidLong() float64 {
	return self.Mid[2] - self.Mid[3]
}
func NewCandles(tmp map[string]interface{}) (c *Candles,err error) {
	c = &Candles{}
	Mid := tmp["mid"].(map[string]interface{})
	if Mid != nil {
		c.Mid[0], err = strconv.ParseFloat(Mid["o"].(string), 64)
		if err != nil {
			return c,err
		}
		c.Mid[1], err = strconv.ParseFloat(Mid["c"].(string), 64)
		if err != nil {
			return c,err
		}
		c.Mid[2], err = strconv.ParseFloat(Mid["h"].(string), 64)
		if err != nil {
			return c,err
		}
		c.Mid[3], err = strconv.ParseFloat(Mid["l"].(string), 64)
		if err != nil {
			return c,err
		}
	}
	c.Volume = tmp["volume"].(float64)
	ti, err := strconv.ParseFloat(tmp["time"].(string), 64)
	if err != nil {
		return c,err
	}
	c.Time = int64(ti)
	return c,nil
}
