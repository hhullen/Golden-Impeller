package t_api

import (
	"context"
	"errors"
	"fmt"
	"time"
	"trading_bot/internal/service"
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

func resolveIntoPbInterval(interval strategy.CandleInterval) pb.CandleInterval {
	if v, ok := intervalMap[interval]; ok {
		return v
	}
	return pb.CandleInterval_CANDLE_INTERVAL_UNSPECIFIED
}

type IStream interface {
	Listen() error
	Stop()
}

type Client struct {
	investgo.Client

	marketDataStream *investgo.MarketDataStream
	ordersDataStream *investgo.OrderStateStream
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

func (c *Client) RecieveLastPrice(instrInfo *datastruct.InstrumentInfo) (*datastruct.LastPrice, error) {
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

	go c.startListeningInstrumentStream(uid, c.marketDataStream)

	c.lastPriceInput = ch

	return nil
}

func (c *Client) startListeningInstrumentStream(uid string, s IStream) {
	for {
		select {
		case <-c.ctx.Done():
			s.Stop()
		default:
			if err := s.Listen(); err != nil {
				c.Logger.Errorf("failed starting listening stream for '%s': %s", uid, err.Error())

				const sleepToListenRetry = 5
				c.Logger.Infof("Sleep for '%s' second", sleepToListenRetry)
				time.Sleep(sleepToListenRetry * time.Second)
			}
		}
	}
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

func (c *Client) MakeSellOrder(instrInfo *datastruct.InstrumentInfo, lots int64, requestId string) (*datastruct.PostOrderResult, error) {
	if lots < 1 {
		return nil, fmt.Errorf("incorrect lots to make order: %d", lots)
	}

	orderResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrInfo.Uid,
		Quantity:     lots,
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

func (c *Client) MakeBuyOrder(instrInfo *datastruct.InstrumentInfo, lots int64, requestId string) (*datastruct.PostOrderResult, error) {
	if lots < 1 {
		return nil, fmt.Errorf("incorrect lots to make order: %d", lots)
	}

	buyResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrInfo.Uid,
		Quantity:     lots,
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

func (c *Client) RecieveOrdersUpdate(instrInfo *datastruct.InstrumentInfo) (*datastruct.Order, error) {
	if c.ordersDataStream == nil {
		stream, err := c.NewOrdersStreamClient().OrderStateStream([]string{c.GetAccoountId()}, 0)
		if err != nil {
			return nil, err
		}
		c.ordersDataStream = stream
		go c.startListeningInstrumentStream(instrInfo.Uid, c.ordersDataStream)
	}

	order, ok := <-c.ordersDataStream.OrderState()
	if !ok {
		return nil, fmt.Errorf("ordersDataStream closed")
	}

	returnable := &datastruct.Order{
		ExecutionReportStatus: order.ExecutionReportStatus.String(),
		OrderPrice: datastruct.Quotation{
			Units: order.OrderPrice.Units,
			Nano:  order.OrderPrice.Nano,
		},
		LotsRequested: order.LotsRequested,
		LotsExecuted:  order.LotsExecuted,
		InstrumentUid: instrInfo.Uid,
	}

	if order.CreatedAt != nil {
		t := order.CreatedAt.AsTime()
		returnable.CreatedAt = &t
	}

	if order.CompletionTime != nil {
		t := order.CompletionTime.AsTime()
		returnable.CompletionTime = &t
	}

	if order.OrderRequestId != nil {
		returnable.OrderId = *order.OrderRequestId
	} else {
		returnable.OrderId = order.OrderId
	}

	if order.Direction == pb.OrderDirection_ORDER_DIRECTION_BUY {
		returnable.Direction = service.Buy.ToString()
	} else {
		returnable.Direction = service.Sell.ToString()
	}

	if order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL ||
		order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL {

		returnable.ExecutionReportStatus = service.Fill.ToString()

	} else if order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED ||
		order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED {

		returnable.ExecutionReportStatus = service.Cancelled.ToString()

	} else {
		returnable.ExecutionReportStatus = service.New.ToString()
	}

	return returnable, nil
}
