package t_api

import (
	"context"
	"errors"
	"fmt"
	"time"
	"trading_bot/internal/service/datastruct"
	"trading_bot/internal/strategy"

	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
)

var intervalMap = map[strategy.CandleInterval]pb.CandleInterval{
	strategy.Interval_1_Min:  pb.CandleInterval_CANDLE_INTERVAL_1_MIN,
	strategy.Interval_5_Min:  pb.CandleInterval_CANDLE_INTERVAL_5_MIN,
	strategy.Interval_15_Min: pb.CandleInterval_CANDLE_INTERVAL_15_MIN,
	strategy.Interval_Hour:   pb.CandleInterval_CANDLE_INTERVAL_HOUR,
	strategy.Interval_Day:    pb.CandleInterval_CANDLE_INTERVAL_DAY,
	strategy.Interval_2_Min:  pb.CandleInterval_CANDLE_INTERVAL_2_MIN,
	strategy.Interval_3_Min:  pb.CandleInterval_CANDLE_INTERVAL_3_MIN,
	strategy.Interval_10_Min: pb.CandleInterval_CANDLE_INTERVAL_10_MIN,
	strategy.Interval_30_Min: pb.CandleInterval_CANDLE_INTERVAL_30_MIN,
	strategy.Interval_2_Hour: pb.CandleInterval_CANDLE_INTERVAL_2_HOUR,
	strategy.Interval_4_Hour: pb.CandleInterval_CANDLE_INTERVAL_4_HOUR,
	strategy.Interval_Week:   pb.CandleInterval_CANDLE_INTERVAL_WEEK,
	strategy.Interval_Month:  pb.CandleInterval_CANDLE_INTERVAL_MONTH,
}

type Client struct {
	investgo.Client

	marketDataStream *investgo.MarketDataStream
	lastPriceInput   <-chan *pb.LastPrice
	ctx              context.Context
}

func NewClient(ctx context.Context, conf investgo.Config, l investgo.Logger) (*Client, error) {
	investClient, err := investgo.NewClient(ctx, conf, l)
	if err != nil {
		return nil, err
	}

	return &Client{Client: *investClient, ctx: ctx}, nil
}

func (c *Client) GetAccoountId() string {
	return c.Config.AccountId
}

func (c *Client) GetLastPrice(instrInfo *datastruct.InstrumentInfo) (*datastruct.LastPrice, error) {
	if c.marketDataStream == nil {

		if err := c.prepareStreamForInstrument(instrInfo.Uid); err != nil {
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

func (c *Client) prepareStreamForInstrument(uid string) error {
	stream, err := c.NewMarketDataStreamClient().MarketDataStream()
	if err != nil {
		return err
	}
	c.marketDataStream = stream

	ch, err := c.marketDataStream.SubscribeLastPrice([]string{uid})
	if err != nil {
		return err
	}

	go c.startListeningInstrumentStream(uid)

	c.lastPriceInput = ch

	return nil
}

// Должен запускаться в отдельной рутине. Блокирующий.
func (c *Client) startListeningInstrumentStream(uid string) {
	for {
		select {
		case <-c.ctx.Done():
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

func (c *Client) GetCandlesHistory(uid string, from, to time.Time, interval strategy.CandleInterval) ([]*datastruct.Candle, error) {
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
	for _, candle := range RespCandles {
		if !candle.IsComplete {
			continue
		}
		candles = append(candles, &datastruct.Candle{
			Timestamp: candle.Time.AsTime(),
			Volume:    candle.Volume,
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
		})
	}

	return candles, nil
}

func resolveIntoPbInterval(interval strategy.CandleInterval) pb.CandleInterval {
	if v, ok := intervalMap[interval]; ok {
		return v
	}
	return pb.CandleInterval_CANDLE_INTERVAL_UNSPECIFIED
}

func (c *Client) GetOrders(uid string) ([]*datastruct.OrderState, error) {
	ordersResp, err := c.NewOrdersServiceClient().GetOrders(c.Config.AccountId)
	if err != nil {
		return nil, err
	}

	orders := []*datastruct.OrderState{}
	for _, order := range ordersResp.Orders {
		if order.InstrumentUid == uid {
			orders = append(orders, &datastruct.OrderState{
				InstrumentUid: order.InstrumentUid,
				OrderId:       order.OrderId,
			})
		}
	}

	return orders, nil
}

func (c *Client) GetPositions(uid string) (*datastruct.Position, error) {
	portfResp, err := c.NewOperationsServiceClient().GetPortfolio(c.GetAccoountId(), pb.PortfolioRequest_RUB)
	if err != nil {
		return nil, err
	}

	for _, pos := range portfResp.PortfolioResponse.Positions {
		if pos.InstrumentUid == uid {

			return &datastruct.Position{
				AveragePositionPrice: datastruct.Quotation{
					Units: pos.AveragePositionPrice.Units,
					Nano:  pos.AveragePositionPrice.Nano,
				},
				Quantity: datastruct.Quotation{
					Units: pos.Quantity.Units,
					Nano:  pos.Quantity.Nano,
				},
			}, nil

		}
	}
	return nil, nil
}

func (c *Client) GetInstrumentInfo(uid string) (*datastruct.InstrumentInfo, error) {
	respInfo, err := c.NewInstrumentsServiceClient().FindInstrument(uid)
	if err != nil {
		return nil, err
	}

	var instrumentInfo *pb.InstrumentShort
	for _, instr := range respInfo.Instruments {
		if instr.Uid == uid {
			instrumentInfo = instr
			break
		}
	}
	if instrumentInfo == nil {
		return nil, fmt.Errorf("not found instrument '%s'", uid)
	}

	return &datastruct.InstrumentInfo{
		Isin:         instrumentInfo.Isin,
		Figi:         instrumentInfo.Figi,
		Ticker:       instrumentInfo.Ticker,
		ClassCode:    instrumentInfo.ClassCode,
		Name:         instrumentInfo.Name,
		Uid:          instrumentInfo.Uid,
		AvailableApi: instrumentInfo.ApiTradeAvailableFlag,
		ForQuals:     instrumentInfo.ForQualInvestorFlag,
		Lot:          instrumentInfo.Lot,
	}, nil
}

func (c *Client) MakeSellOrder(instrInfo *datastruct.InstrumentInfo, quantity int64, requestId string) (*datastruct.PostOrderResult, error) {
	if !isQuantityCorrect(quantity, int64(instrInfo.Lot)) {
		return nil, fmt.Errorf("incorrect quantity to make order: %d", quantity/int64(instrInfo.Lot))
	}

	orderResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrInfo.Uid,
		Quantity:     quantity,
		Direction:    pb.OrderDirection_ORDER_DIRECTION_SELL,
		AccountId:    c.GetAccoountId(),
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
		OrderId:      requestId,
	})
	if err != nil {
		return nil, err
	}

	return &datastruct.PostOrderResult{
		ExecutedCommission: datastruct.Quotation{
			Units: orderResp.ExecutedCommission.Units,
			Nano:  orderResp.ExecutedCommission.Nano,
		},
		ExecutedOrderPrice: datastruct.Quotation{
			Units: orderResp.ExecutedOrderPrice.Units,
			Nano:  orderResp.ExecutedOrderPrice.Nano,
		},
		InstrumentUid:         orderResp.InstrumentUid,
		OrderId:               orderResp.OrderId,
		ExecutionReportStatus: orderResp.ExecutionReportStatus.String(),
	}, nil
}

func (c *Client) MakeBuyOrder(instrInfo *datastruct.InstrumentInfo, quantity int64, requestId string) (*datastruct.PostOrderResult, error) {
	if !isQuantityCorrect(quantity, int64(instrInfo.Lot)) {
		return nil, fmt.Errorf("incorrect quantity to make order: %d", quantity/int64(instrInfo.Lot))
	}

	buyResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrInfo.Uid,
		Quantity:     quantity,
		Direction:    pb.OrderDirection_ORDER_DIRECTION_BUY,
		AccountId:    c.GetAccoountId(),
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
		OrderId:      requestId,
	})
	if err != nil {
		return nil, err
	}

	return &datastruct.PostOrderResult{
		ExecutedCommission: datastruct.Quotation{
			Units: buyResp.ExecutedCommission.Units,
			Nano:  buyResp.ExecutedCommission.Nano,
		},
		ExecutedOrderPrice: datastruct.Quotation{
			Units: buyResp.ExecutedOrderPrice.Units,
			Nano:  buyResp.ExecutedOrderPrice.Nano,
		},
		InstrumentUid:         buyResp.InstrumentUid,
		OrderId:               buyResp.OrderId,
		ExecutionReportStatus: buyResp.ExecutionReportStatus.String(),
	}, nil
}

func isQuantityCorrect(quantity, lot int64) bool {
	return quantity/lot > 0
}
