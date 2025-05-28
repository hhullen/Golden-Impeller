package t_api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ds "trading_bot/internal/service/datastruct"
	"trading_bot/internal/supports"

	"github.com/google/uuid"
	"github.com/russianinvestments/invest-api-go-sdk/investgo"
	pb "github.com/russianinvestments/invest-api-go-sdk/proto"
)

var intervalMap = map[ds.CandleInterval]pb.CandleInterval{
	ds.Interval_1_Min:  pb.CandleInterval_CANDLE_INTERVAL_1_MIN,
	ds.Interval_5_Min:  pb.CandleInterval_CANDLE_INTERVAL_5_MIN,
	ds.Interval_15_Min: pb.CandleInterval_CANDLE_INTERVAL_15_MIN,
	ds.Interval_Hour:   pb.CandleInterval_CANDLE_INTERVAL_HOUR,
	ds.Interval_Day:    pb.CandleInterval_CANDLE_INTERVAL_DAY,
	ds.Interval_2_Min:  pb.CandleInterval_CANDLE_INTERVAL_2_MIN,
	ds.Interval_3_Min:  pb.CandleInterval_CANDLE_INTERVAL_3_MIN,
	ds.Interval_10_Min: pb.CandleInterval_CANDLE_INTERVAL_10_MIN,
	ds.Interval_30_Min: pb.CandleInterval_CANDLE_INTERVAL_30_MIN,
	ds.Interval_2_Hour: pb.CandleInterval_CANDLE_INTERVAL_2_HOUR,
	ds.Interval_4_Hour: pb.CandleInterval_CANDLE_INTERVAL_4_HOUR,
	ds.Interval_Week:   pb.CandleInterval_CANDLE_INTERVAL_WEEK,
	ds.Interval_Month:  pb.CandleInterval_CANDLE_INTERVAL_MONTH,
}

func resolveIntoPbInterval(interval ds.CandleInterval) pb.CandleInterval {
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
	sync.RWMutex
	investgo.Client

	marketDataStream *investgo.MarketDataStream
	ordersDataStream *investgo.OrderStateStream
	lastPriceInput   map[string]map[uuid.UUID]chan *pb.LastPrice
	ordersStateInput map[string]map[string]map[uuid.UUID]chan *pb.OrderStateStreamResponse_OrderState
	ctx              context.Context
}

func NewClient(ctx context.Context, conf investgo.Config, l investgo.Logger) (*Client, error) {
	investClient, err := investgo.NewClient(ctx, conf, l)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Client:           *investClient,
		ctx:              ctx,
		lastPriceInput:   make(map[string]map[uuid.UUID]chan *pb.LastPrice),
		ordersStateInput: make(map[string]map[string]map[uuid.UUID]chan *pb.OrderStateStreamResponse_OrderState),
	}

	return c, nil
}

func (c *Client) RegisterLastPriceRecipient(instrInfo *ds.InstrumentInfo) error {
	if c.marketDataStream == nil {
		if err := c.prepareStreamForInstrument(instrInfo); err != nil {
			return err
		}
	} else {
		_, err := c.marketDataStream.SubscribeLastPrice([]string{instrInfo.Uid})
		if err != nil {
			return err
		}
	}

	c.Lock()
	if _, ok := c.lastPriceInput[instrInfo.Uid]; !ok {
		c.lastPriceInput[instrInfo.Uid] = make(map[uuid.UUID]chan *pb.LastPrice)
	}
	if _, ok := c.lastPriceInput[instrInfo.Uid][instrInfo.InstanceId]; !ok {
		c.lastPriceInput[instrInfo.Uid][instrInfo.InstanceId] = make(chan *pb.LastPrice)
	}
	c.Unlock()

	return nil
}

func (c *Client) RecieveLastPrice(instrInfo *ds.InstrumentInfo) (*ds.LastPrice, error) {
	c.RLock()
	ch := c.lastPriceInput[instrInfo.Uid][instrInfo.InstanceId]
	c.RUnlock()

	lastPrice, ok := <-ch
	if !ok {
		return nil, errors.New("marketDataStream closed")
	}

	return &ds.LastPrice{
		Price: ds.Quotation{
			Units: lastPrice.Price.Units,
			Nano:  lastPrice.Price.Nano,
		},
		Time: lastPrice.Time.AsTime(),
		Uid:  lastPrice.InstrumentUid,
		Figi: lastPrice.Figi,
	}, nil

}

func (c *Client) prepareStreamForInstrument(instrInfo *ds.InstrumentInfo) error {
	stream, err := c.NewMarketDataStreamClient().MarketDataStream()
	if err != nil {
		return err
	}
	c.marketDataStream = stream

	ch, err := c.marketDataStream.SubscribeLastPrice([]string{instrInfo.Uid})
	if err != nil {
		return err
	}

	go c.startInstrumenstRouting(ch)

	go c.startListeningInstrumentStream(instrInfo.Uid, c.marketDataStream)

	return nil
}

func (c *Client) startInstrumenstRouting(ch <-chan *pb.LastPrice) {
	defer func() {
		for _, v := range c.lastPriceInput {
			for _, ch := range v {
				close(ch)
			}
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case v, ok := <-ch:
			if !ok {
				return
			}

			c.Lock()
			for _, uniqueListener := range c.lastPriceInput[v.InstrumentUid] {
				select {
				case uniqueListener <- v:
				default:
				}
			}
			c.Unlock()
		}
	}
}

func (c *Client) startListeningInstrumentStream(uid string, s IStream) {
	for {
		select {
		case <-c.ctx.Done():
			s.Stop()
		default:
			if err := s.Listen(); err != nil {
				c.Logger.Errorf("failed starting listening stream for '%s': %s", uid, err.Error())

				const sleepToListenRetry = time.Second * 5
				c.Logger.Infof("Sleep for '%s' second", sleepToListenRetry)
				supports.WaitFor(c.ctx, sleepToListenRetry)
			}
		}
	}
}

func (c *Client) GetInstrumentInfo(uid string) (*ds.InstrumentInfo, error) {
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

	return &ds.InstrumentInfo{
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

func (c *Client) MakeSellOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, accountId string) (*ds.PostOrderResult, error) {
	if lots < 1 {
		return nil, fmt.Errorf("incorrect lots to make order: %d", lots)
	}

	orderResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrInfo.Uid,
		Quantity:     lots,
		Direction:    pb.OrderDirection_ORDER_DIRECTION_SELL,
		AccountId:    accountId,
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
		OrderId:      requestId,
	})
	if err != nil {
		return nil, err
	}

	return &ds.PostOrderResult{
		ExecutedCommission: ds.Quotation{
			Units: orderResp.ExecutedCommission.Units,
			Nano:  orderResp.ExecutedCommission.Nano,
		},
		ExecutedOrderPrice: ds.Quotation{
			Units: orderResp.ExecutedOrderPrice.Units,
			Nano:  orderResp.ExecutedOrderPrice.Nano,
		},
		InstrumentUid:         orderResp.InstrumentUid,
		OrderId:               orderResp.OrderId,
		ExecutionReportStatus: orderResp.ExecutionReportStatus.String(),
	}, nil
}

func (c *Client) MakeBuyOrder(instrInfo *ds.InstrumentInfo, lots int64, requestId, accountId string) (*ds.PostOrderResult, error) {
	if lots < 1 {
		return nil, fmt.Errorf("incorrect lots to make order: %d", lots)
	}

	buyResp, err := c.NewOrdersServiceClient().PostOrder(&investgo.PostOrderRequest{
		InstrumentId: instrInfo.Uid,
		Quantity:     lots,
		Direction:    pb.OrderDirection_ORDER_DIRECTION_BUY,
		AccountId:    accountId,
		OrderType:    pb.OrderType_ORDER_TYPE_BESTPRICE,
		OrderId:      requestId,
	})
	if err != nil {
		return nil, err
	}

	return &ds.PostOrderResult{
		ExecutedCommission: ds.Quotation{
			Units: buyResp.ExecutedCommission.Units,
			Nano:  buyResp.ExecutedCommission.Nano,
		},
		ExecutedOrderPrice: ds.Quotation{
			Units: buyResp.ExecutedOrderPrice.Units,
			Nano:  buyResp.ExecutedOrderPrice.Nano,
		},
		InstrumentUid:         buyResp.InstrumentUid,
		OrderId:               buyResp.OrderId,
		ExecutionReportStatus: buyResp.ExecutionReportStatus.String(),
	}, nil
}

func (c *Client) RegisterOrderStateRecipient(instrInfo *ds.InstrumentInfo, accountId string) error {
	if c.ordersDataStream == nil {
		err := c.prepareStreamForOrdersState(instrInfo)
		if err != nil {
			return err
		}
	}

	c.Lock()
	if _, ok := c.ordersStateInput[accountId]; !ok {
		c.ordersStateInput[accountId] = make(map[string]map[uuid.UUID]chan *pb.OrderStateStreamResponse_OrderState)
	}
	if _, ok := c.ordersStateInput[accountId][instrInfo.Uid]; !ok {
		c.ordersStateInput[accountId][instrInfo.Uid] = make(map[uuid.UUID]chan *pb.OrderStateStreamResponse_OrderState)
	}
	if _, ok := c.ordersStateInput[accountId][instrInfo.Uid][instrInfo.InstanceId]; !ok {
		c.ordersStateInput[accountId][instrInfo.Uid][instrInfo.InstanceId] = make(chan *pb.OrderStateStreamResponse_OrderState)
	}
	c.Unlock()

	return nil
}

func (c *Client) RecieveOrdersUpdate(instrInfo *ds.InstrumentInfo, accountId string) (*ds.Order, error) {
	c.RLock()
	ch := c.ordersStateInput[accountId][instrInfo.Uid][instrInfo.InstanceId]
	c.RUnlock()

	order, ok := <-ch
	if !ok {
		return nil, fmt.Errorf("ordersDataStream closed")
	}

	returnable := &ds.Order{
		ExecutionReportStatus: order.ExecutionReportStatus.String(),
		OrderPrice: ds.Quotation{
			Units: order.OrderPrice.Units,
			Nano:  order.OrderPrice.Nano,
		},
		LotsRequested: order.LotsRequested,
		LotsExecuted:  order.LotsExecuted,
		InstrumentUid: order.InstrumentUid,
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
		returnable.Direction = ds.Buy.ToString()
	} else {
		returnable.Direction = ds.Sell.ToString()
	}

	if order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_FILL ||
		order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_PARTIALLYFILL {

		returnable.ExecutionReportStatus = ds.Fill.ToString()

	} else if order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_REJECTED ||
		order.ExecutionReportStatus == pb.OrderExecutionReportStatus_EXECUTION_REPORT_STATUS_CANCELLED {

		returnable.ExecutionReportStatus = ds.Cancelled.ToString()

	} else {
		returnable.ExecutionReportStatus = ds.New.ToString()
	}

	return returnable, nil
}

func (c *Client) prepareStreamForOrdersState(instrInfo *ds.InstrumentInfo) error {
	stream, err := c.NewOrdersStreamClient().OrderStateStream([]string{}, 0)
	if err != nil {
		return err
	}

	c.ordersDataStream = stream

	go c.startOrdersStateRouting(instrInfo, c.ordersDataStream.OrderState())

	go c.startListeningInstrumentStream(instrInfo.Uid, c.ordersDataStream)

	return nil
}

func (c *Client) startOrdersStateRouting(instrInfo *ds.InstrumentInfo, ch <-chan *pb.OrderStateStreamResponse_OrderState) {
	defer func() {
		for _, byAccId := range c.ordersStateInput {
			for _, byUID := range byAccId {
				for _, ch := range byUID {
					close(ch)
				}
			}
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case v, ok := <-ch:
			if !ok {
				return
			}

			c.Lock()
			for _, uniqueListener := range c.ordersStateInput[v.AccountId][v.InstrumentUid] {
				select {
				case uniqueListener <- v:
				default:
				}
			}
			c.Unlock()
		}
	}
}
