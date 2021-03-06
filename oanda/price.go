package oanda
import(
	"strconv"
	//"time"
	//"strings"
)
type Price struct {
	Type string `json:"type"`
	Instrument InstrumentName `json:"instrument"`
	Time DateTime `json:"time"`
	Status string `json:"status"`
	Tradeable bool `json:"tradeable"`
	Bids []*PriceBucket `json:"bids"`
	Asks []*PriceBucket `json:"asks"`
	CloseoutBid PriceValue `json:"closeoutBid"`
	CloseoutAsk PriceValue `json:"closeoutAsk"`
	QuoteHomeConversionFactors QuoteHomeConversionFactors `json:"quoteHomeConversionFactors"`
	UnitsAvailable UnitsAvailable `json:"unitsAvailable"`
}
func (self *Price) Bid() float64 {
	return self.bid()
}
func (self *Price) bid() float64 {
	bid ,err := strconv.ParseFloat(string(self.CloseoutBid),64)
	if err != nil {
		panic(err)
	}
	return bid
}
func (self *Price) Ask() float64 {
	return self.ask()
}
func (self *Price) ask() float64 {
	ask ,err := strconv.ParseFloat(string(self.CloseoutAsk),64)
	if err != nil {
		panic(err)
	}
	return ask
}

func (self *Price) Diff() float64 {
	return self.diff()
}
func (self *Price) diff() float64 {
	return self.ask() - self.bid()
}
func (self *Price) Middle() float64 {
	return self.middle()
}
func (self *Price) middle() float64 {
	return (self.ask() + self.bid())/2
}

