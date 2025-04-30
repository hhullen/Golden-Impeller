package strategy

type CandleInterval int32

const (
	Interval_1_Min CandleInterval = iota
	Interval_5_Min
	Interval_15_Min
	Interval_Hour
	Interval_Day
	Interval_2_Min
	Interval_3_Min
	Interval_10_Min
	Interval_30_Min
	Interval_2_Hour
	Interval_4_Hour
	Interval_Week
	Interval_Month
)
