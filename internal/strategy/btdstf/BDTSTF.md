# BDTSTF

Main strategy idea is selling when there is certain of percentage profit of specific buy order reached. Every buy order make when price goes down and sell when price getting up.

```mermaid
graph TD
    A[Got last price] --> B{ Get last buy order with lowest price };
    B -- exists --> C{ Is price down on certain percent to buy };
    C -- no --> F{ Is price up on certain percent to down and there is any buy orders };
    C -- yes --> I;
    B -- not --> D{ Get last sell order };
    D -- exists --> I{ Is buy orders higher than max depth parameter };
    I -- no --> J;
    I -- yes --> K{ Get higher buy order which was not sold };
    K -- exists --> L[ Sell higher buy order ];
    K -- not --> J;
    L --> J;
    D -- not --> J[ Buy ];
    F -- yes --> G[ Sell ];
    F -- yes --> H[ Hold ];
```

Here are required parameters for `strategy_cfg` section.
* `name` must be in every config. This is how certain strategy is resolved while trader is starting
* `max_depth` is a maximum amount of buy orders are unbalanced by sell orders
* `lots_to_buy` lots to buy in one order
* `percent_down_to_buy` is a percent on which price should be down to buy.  
`!`Not fraction but true percent value. For example if 1.65% needed, use 1.65 not 0.0165.
* `percent_up_to_sell` is a percent on which price should be up to sell.
