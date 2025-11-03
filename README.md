# GOLDEN IMPELLER

Here is a trading bot coded on golang using T-Invest API  
[T-Invest Intro](https://developer.tbank.ru/invest/intro/intro)  
[API T-Invest](https://developer.tbank.ru/invest/api)  
[Golang SDK](https://github.com/RussianInvestments/invest-api-go-sdk)

## Requirements
GNU Make 3.81, go1.24.2, Docker (Docker Desktop)
  
# Before the start
Create a `.env.yaml` file with configuration example below
```yaml
APP_NAME: Golden_Impeller

T_INVEST_TOKEN: <your token>
T_INVEST_ADDRESS: <endpoint>
T_INVEST_ACCOUNT_ID: <default account id>

TRADER:
    trading_delay: 300ms
    on_trading_error_delay: 10s
    on_orders_operating_error_delay: 10s

    traders:
        - unique_trader_id: tgld_[btdstf-4-100-0.75-1.7]
          uid: de82be66-3b9b-4612-9572-61e3c6039013
          account_id: <account id>
          strategy_cfg:
            name: btdstf
            max_depth: 4
            lots_to_buy: 100
            percent_down_to_buy: 0.75
            percent_up_to_sell: 1.7

BACKTESTER:
    - unique_trader_id: tgld_san_22_0.75_1.7
      uid: de82be66-3b9b-4612-9572-61e3c6039013
      from: 01.01.2023
      to: now
      interval: 1min
      start_deposit: 20000
      commission_percent: 0.0
      strategy_cfg:
        name: btdstf
        max_depth: 4
        lots_to_buy: 100
        percent_down_to_buy: 0.75
        percent_up_to_sell: 1.7


HISTORY_CANDLES_LOADER:
    - ticker: TGLD
    uid: de82be66-3b9b-4612-9572-61e3c6039013
    from: 01.01.2020
    to: now
    interval: 1min
```
`!` Uid of instruments can be changed fromt time to time

Let's brake config donw step by step:
* `APP_NAME` can take any value
* `T_INVEST_TOKEN` should take your T-Invest token. [How to create new one](https://developer.tbank.ru/invest/intro/intro/token)
* `T_INVEST_ADDRESS` can take prod endpoint `invest-public-api.tbank.ru:443` where orders places on real account with real money or `sandbox-invest-public-api.tbank.ru:443` which is sandbox
* `T_INVEST_ACCOUNT_ID` can take id of one your account where orders will be placed. This variavle is used as a default account if another not specified for certain "trader".

* `TRADER` is a Trades Service and take some common settings and list of "traders". Trader Service starts every "trader" in loop where it gets new price, gives price to strategy object which returns list of actions, and trying to execute these actions. In parallel with this loop exists loop for operating orders. When "trader" placed any order API T-Invest send report that operates in orders operating loop.
    * `trading_delay` is a delay for common loop of "traders"
    * `on_trading_error_delay` is a delay when some error was occured on getting price or getting strategy actions or executing orders.
    * `on_orders_operating_error_delay` is a delay when some error in orders operating loop
    * `traders` is a list of "traders". Every trader require next fields:
        * `unique_trader_id` that must be unique among of traders
        * `uid` that is uid of certain instrument
        * `account_id` if it is needed to set different account id for certain instrument rather than default
        * `strategy_cfg` contains a map of parameters for certain Strategy. Could be found in Strategy description. Here are parameters for some Strategy implemented as example. [btdstf description](./internal/strategy/btdstf/BDTSTF.md)

* `BACKTESTER` is a list of configs for "backtesters" to launch back test on history data for some strategy. Here are required fields:
    * `unique_trader_id` the same as unique id for trader. Can take any values but unique among of "backtesters"
    * `uid` that is uid of certain instrument
    * `from` take a date where to start a backtest. Can take formats:
        * "2006-01-02"  
		* "2006/01/02"
		* "2006.01.02"
		* "02-01-2006"
		* "02.01.2006"
		* "02/01/2006"
        * 12 - just a number as a months ago
    * `to` take a date where to end a backtest. Can take the same formats as 'from' field and additionaly `now` value
    * `interval` is a candeles interval for test. `!`But these candles have to be loaded before start testing. How to load will be described next.
    * `start_deposit` takes a number of rubles deposit for test
    * `commission_percent` is a commision of every order
    * `strategy_cfg` as well as for trader described above

* `HISTORY_CANDLES_LOADER` is a list of configs fo loading candles for backtest
    * `ticker` is a ticker for instrument
    * `uid` that is uid of certain instrument
    * `from`, `to` as well as described above
    * `interval` is a candles interval to load. Can take next values:
        * 1min
		* 2min
		* 3min
		* 5min
		* 10min
		* 15min
		* 30min
		* 1hour
		* 2hour
		* 4hour
		* 1day
		* 1week
		* 1month

# How to start the Backtest
When `T_INVEST_TOKEN`, `T_INVEST_ADDRESS` and `T_INVEST_ACCOUNT_ID` filled.  

1. Make `tools` binary. It creates ./cmd/tools/tools or ./cmd/tools/tools.exe on Windows
```
make tools
```

2. Make account. When using sandbox.
```
./cmd/tools/tools create-sandbox-account
```
 It will show message:
```
New account: a15aa0bd-557c-40ed-a15e-0731a66b5e70
```
It can be possible to get opened accounts into file accounts.ayml
```
./cmd/tools/tools get-accounts
```  
`!` When using real account it must be opened on T-Invest and token must be created for this account.  
To close account:
```
./cmd/tools/tools close-sandbox-account <account id>
```


3. Get instruments: 
```
make get-instruments
```
It creates files bonds.txt, currencies.txt, etfs.txt and shares.txt.  
Getting uid and ticker from file make `HISTORY_CANDLES_LOADER` config. For example:
```yaml
HISTORY_CANDLES_LOADER:
    - ticker: SBER
    uid: e6123145-9665-43e0-8413-cd61b8aa9b13
    from: 01-01-2024
    to: now
    interval: 1min

    - ticker: TGLD
    uid: de82be66-3b9b-4612-9572-61e3c6039013
    from: 01.01.2023
    to: now
    interval: 1min
```

4. Start local database and make migrations. It is require `Docker` installed.  
On windows opend `Docker desktop` is required.  
Default credectials for PostreSQL placed in `./secrets`. 
```
make start-local-database
make migrations-up
```
To stop database and remove image
```
make stop-local-database
```

5. Load candles for instrament
```
make load-candles
```
It tries requesting in a rarely way, but sometimes API could send message `INFO Resource Exhausted, sleep for Xs...` due-to many requests in time. It will not stop loading candles. Just wait.

6. Fill a BACKTESTER config. For example:
```yaml
BACKTESTER:
    - unique_trader_id: sber_chan_24_0.75_1.7
      uid: e6123145-9665-43e0-8413-cd61b8aa9b13
      from: 01.01.2024
      to: now
      interval: 1min
      start_deposit: 100000
      commission_percent: 0.4
      strategy_cfg:
        name: btdstf
        max_depth: 5
        lots_to_buy: 1
        percent_down_to_buy: 0.75
        percent_up_to_sell: 1.7

    - unique_trader_id: tgld_san_22_0.75_1.7
      uid: de82be66-3b9b-4612-9572-61e3c6039013
      from: 01.01.2024
      to: now
      interval: 1min
      start_deposit: 20000
      commission_percent: 0.0
      strategy_cfg:
        name: btdstf
        max_depth: 4
        lots_to_buy: 100
        percent_down_to_buy: 0.75
        percent_up_to_sell: 1.7
```

7. Start backtest
```
make backtest
```
Here could be some delay before starting after command entered due-to running backtest as fast as it can be by loading candles in RAM.

# How to start Trader Service Locally
When `T_INVEST_TOKEN`, `T_INVEST_ADDRESS` and `T_INVEST_ACCOUNT_ID` filled.  
1. First look at 1-4 points in [How to start the Backtest](#How-to-start-the-Backtest)

2. When using sandbox account topup balance. This command adds value to balance.
```
./cmd/tools/tools topup-sandbox-account <account id> <value>
```
To set new balance value use:
```
./cmd/tools/tools setup-sandbox-account <account id> <value>
```

3. Fill `TRADER` config. For example:
```yaml
TRADER:
    trading_delay: 300ms
    on_trading_error_delay: 10s
    on_orders_operating_error_delay: 10s

    traders:
        - unique_trader_id: tgld_[btdstf-4-100-0.75-1.7]
          uid: de82be66-3b9b-4612-9572-61e3c6039013
          account_id: <your another account id>
          strategy_cfg:
            name: btdstf
            max_depth: 4
            lots_to_buy: 100
            percent_down_to_buy: 0.75
            percent_up_to_sell: 1.7

        - unique_trader_id: sber_[btdstf-5-1-0.75-1.7]
          uid: e6123145-9665-43e0-8413-cd61b8aa9b13
          strategy_cfg:
            name: btdstf
            max_depth: 5
            lots_to_buy: 1
            percent_down_to_buy: 0.75
            percent_up_to_sell: 1.7
```

4. start local trader
```
make local-trader
```
When started, the current terminal shows only orders any "trater" made. Technical logs from API and Trading Manager could be found in `invest.log` and `trading_manager.log` files. It is convenient to look at these log files in other terminal with command:
```
# For Linux
tail -n 200 -f ./invest.log
tail -n 200 -f ./trading_manager.log

# For Windows Powershell
Get-Content -Path ".\invest.log" -Tail 200 -Wait
Get-Content -Path ".\trading_manager.log" -Tail 200 -Wait
```

# How to start Trader Service in Docker Compose
When `T_INVEST_TOKEN`, `T_INVEST_ADDRESS` and `T_INVEST_ACCOUNT_ID` filled.  
1. First look at 1-3 points in [How to start Trader Service Locally](#How-to-start-Trader-Service-Locally)

2. Run Trader Service in Docker Compose
```
make trader
```
Here all logs are wroten in stdout and to Clickhouse database.  
The logs and some charts are accessible in Grafana dashboard named `Golden Impeller` via browser:
```
http://localhost:3000/dashboards
```
Default username `admin` and password `admin`.  
In case `.env.yaml` has some changes after Trader Service started, update traders with command:
```
make update-traders-config
```

# Tools options
* Create sandbox account
```
./cmd/tools/tools create-sandbox-account
```

* Close sandbox account
```
./cmd/tools/tools close-sandbox-account  <account id>
```

* Get opened accounts
```
./cmd/tools/tools get-accounts
```

* Get instruments
```
./cmd/tools/tools get-instruments
```

* Topup sandbox account balance
```
./cmd/tools/tools topup-sandbox-account <account id> <value>
```

* Setup sandbox account balance
```
./cmd/tools/tools setup-sandbox-account <account id> <value>
```

* Sell instrument
```
./cmd/tools/tools sell <account id> <instrument uid> <lots>
```

* Buy instrument
```
./cmd/tools/tools buy <account id> <instrument uid> <lots>
```

# Makefile targets
* Generate mocks for interfaces
```
make generate-mocks
```

* Start local database, make migrations, get accounts and get instruments
```
make start
```

* Make tool binary
```
make tools
```

* Make migrations
```
make migrations-up
```

* Remove migrations
```
make migrations-down
```

* Check migrations
```
make migrations-status
```

* Start Backtest
```
make backtest
```

* Load candles
```
make load-candles
```

* Get file with accounts
```
make get-accounts
```

* Get instruments files
```
make get-instruments
```

* Update .env.yaml config when Trader Service is running in Docker Compose
```
make update-traders-config
```

* Start local PostgreSQL databade in docker
```
make start-local-database
```

* Stop running local database
```
make stop-local-database
```

* Run local Trader Service
```
make local-trader
```

* Run Trader Service in Docker Compose
```
make trader
```
* Stop Trader Service in Docker Compose
```
make stop-trader
```
* Recreate images and run Trader Service in Docker Compose. `!` This cleans docker volumes
```
make trader-rebuild
```

* Clean built binaries
```
make clean
```