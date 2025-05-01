package t_api

import (
	"context"
	"errors"
	"time"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/strategy"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
)

const (
	gettingCandlesLimit = 5000
)

type Client struct {
	investgo.Client

	marketDataStream *investgo.MarketDataStream
	lastPriceInput   <-chan *pb.LastPrice
}

func NewClient(ctx context.Context, conf investgo.Config, l investgo.Logger) (*Client, error) {
	investClient, err := investgo.NewClient(ctx, conf, l)
	if err != nil {
		return nil, err
	}

	return &Client{Client: *investClient}, nil
}

func (c *Client) GetLastPrice(ctx context.Context, uid string) (*datastruct.LastPrice, error) {
	if c.marketDataStream == nil {

		if err := c.prepareStreamForInstrument(ctx, uid); err != nil {
			return nil, err
		}

	}

	lastPrice, ok := <-c.lastPriceInput
	if !ok {
		return nil, errors.New("stream closed")
	}

	return &datastruct.LastPrice{
		Price: datastruct.Quotation{
			Units: lastPrice.Price.Units,
			Nano:  lastPrice.Price.Nano,
		},
		Time: lastPrice.Time.AsTime(),
		Uid:  lastPrice.InstrumentUid,
		Figi: lastPrice.Figi,
	}, nil

}

func (c *Client) prepareStreamForInstrument(ctx context.Context, uid string) error {
	stream, err := c.NewMarketDataStreamClient().MarketDataStream()
	if err != nil {
		return err
	}
	c.marketDataStream = stream

	ch, err := c.marketDataStream.SubscribeLastPrice([]string{uid})
	if err != nil {
		return err
	}

	go c.startListeningInstrumentStream(ctx, uid)

	c.lastPriceInput = ch

	return nil
}

// Должен запускаться в отдельной рутине. Блокирующий.
func (c *Client) startListeningInstrumentStream(ctx context.Context, uid string) {
	for {
		select {
		case <-ctx.Done():
			c.marketDataStream.Stop()
		default:
			if err := c.marketDataStream.Listen(); err != nil {
				c.Logger.Errorf("failed starting listening stream for '%s': %s", uid, err.Error())

				const sleepToListenRetry = 5
				c.Logger.Infof("Sleep for '%s' second", sleepToListenRetry)
				time.Sleep(sleepToListenRetry * time.Second)
			}
		}
	}
}

func (c *Client) GetCandlesHistory(ctx context.Context, uid string, from, to time.Time, interval strategy.CandleInterval) ([]*datastruct.Candle, error) {
	// r, _ := c.NewMarketDataServiceClient().GetCandles(uid, resolveIntoPbInterval(interval), from, to, pb.GetCandlesRequest_CANDLE_SOURCE_EXCHANGE, gettingCandlesLimit)

	RespCandles, err := c.NewMarketDataServiceClient().GetHistoricCandles(&investgo.GetHistoricCandlesRequest{
		Instrument: uid,
		Interval:   resolveIntoPbInterval(interval),
		From:       from,
		To:         to,
		Source:     pb.GetCandlesRequest_CANDLE_SOURCE_EXCHANGE,
	})
	if err != nil {
		return nil, err
	}

	candles := make([]*datastruct.Candle, 0, len(RespCandles))
	for i := range RespCandles {
		candles = append(candles, &datastruct.Candle{
			Time:   RespCandles[i].Time.AsTime(),
			Volume: RespCandles[i].Volume,
			Open: datastruct.Quotation{
				Units: RespCandles[i].Open.Units,
				Nano:  RespCandles[i].Open.Nano,
			},
			Close: datastruct.Quotation{
				Units: RespCandles[i].Close.Units,
				Nano:  RespCandles[i].Close.Nano,
			},
			High: datastruct.Quotation{
				Units: RespCandles[i].High.Units,
				Nano:  RespCandles[i].High.Nano,
			},
			Low: datastruct.Quotation{
				Units: RespCandles[i].Low.Units,
				Nano:  RespCandles[i].Low.Nano,
			},
		})
	}

	return candles, nil
}

func resolveIntoPbInterval(interval strategy.CandleInterval) pb.CandleInterval {
	switch interval {
	case strategy.Interval_1_Min:
		return pb.CandleInterval_CANDLE_INTERVAL_1_MIN
	case strategy.Interval_5_Min:
		return pb.CandleInterval_CANDLE_INTERVAL_5_MIN
	case strategy.Interval_15_Min:
		return pb.CandleInterval_CANDLE_INTERVAL_15_MIN
	case strategy.Interval_Hour:
		return pb.CandleInterval_CANDLE_INTERVAL_HOUR
	case strategy.Interval_Day:
		return pb.CandleInterval_CANDLE_INTERVAL_DAY
	case strategy.Interval_2_Min:
		return pb.CandleInterval_CANDLE_INTERVAL_2_MIN
	case strategy.Interval_3_Min:
		return pb.CandleInterval_CANDLE_INTERVAL_3_MIN
	case strategy.Interval_10_Min:
		return pb.CandleInterval_CANDLE_INTERVAL_10_MIN
	case strategy.Interval_30_Min:
		return pb.CandleInterval_CANDLE_INTERVAL_30_MIN
	case strategy.Interval_2_Hour:
		return pb.CandleInterval_CANDLE_INTERVAL_2_HOUR
	case strategy.Interval_4_Hour:
		return pb.CandleInterval_CANDLE_INTERVAL_4_HOUR
	case strategy.Interval_Week:
		return pb.CandleInterval_CANDLE_INTERVAL_WEEK
	case strategy.Interval_Month:
		return pb.CandleInterval_CANDLE_INTERVAL_MONTH
	default:
		return pb.CandleInterval_CANDLE_INTERVAL_UNSPECIFIED
	}
}

func (c *Client) GetOrders(ctx context.Context, uid string) ([]*datastruct.OrderState, error) {
	ordersResp, err := c.Client.NewOrdersServiceClient().GetOrders(c.Config.AccountId)
	if err != nil {
		return nil, err
	}

	orders := []*datastruct.OrderState{}
	for i := range ordersResp.Orders {
		if ordersResp.Orders[i].InstrumentUid == uid {
			orders = append(orders, &datastruct.OrderState{
				InstrumentUid: ordersResp.Orders[i].InstrumentUid,
				OrderId:       ordersResp.Orders[i].OrderId,
			})
		}
	}

	return orders, nil
}
