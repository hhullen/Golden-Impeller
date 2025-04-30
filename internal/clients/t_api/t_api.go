package t_api

import (
	"context"
	"errors"
	"time"
	"trading_bot/internal/service/datastruct"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
)

type Client struct {
	investgo.Client

	marketDataStream *investgo.MarketDataStream
	candlesInput     <-chan *pb.Candle
}

func NewClient(ctx context.Context, conf investgo.Config, l investgo.Logger) (*Client, error) {
	investClient, err := investgo.NewClient(ctx, conf, l)
	if err != nil {
		return nil, err
	}

	return &Client{Client: *investClient}, nil
}

func (c *Client) GetLastCandle(ctx context.Context, uid string) (*datastruct.Candle, error) {
	if c.marketDataStream == nil {

		if err := c.prepareStreamForInstrument(ctx, uid); err != nil {
			return nil, err
		}

	}

	candle, ok := <-c.candlesInput
	if !ok {
		return nil, errors.New("stream closed")
	}

	return &datastruct.Candle{
		Uid:  uid,
		Figi: candle.Figi,
		Open: datastruct.Quotation{
			Units: candle.Open.Units,
			Nano:  candle.Open.Nano,
		},
		Close: datastruct.Quotation{
			Units: candle.Close.Units,
			Nano:  candle.Close.Nano,
		},
		High: datastruct.Quotation{
			Units: candle.High.Units,
			Nano:  candle.High.Nano,
		},
		Low: datastruct.Quotation{
			Units: candle.Low.Units,
			Nano:  candle.Low.Nano,
		},
	}, nil

}

func (c *Client) prepareStreamForInstrument(ctx context.Context, uid string) error {
	stream, err := c.NewMarketDataStreamClient().MarketDataStream()
	if err != nil {
		return err
	}
	c.marketDataStream = stream

	ch, err := c.marketDataStream.SubscribeCandle(
		[]string{uid},
		pb.SubscriptionInterval_SUBSCRIPTION_INTERVAL_ONE_MINUTE,
		true,
		pb.GetCandlesRequest_CANDLE_SOURCE_EXCHANGE.Enum(),
	)
	if err != nil {
		return err
	}

	go c.startListeningInstrumentStream(ctx, uid)

	c.candlesInput = ch

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
