# ElevatorPractise

以 Go 語言實作的電梯模擬系統，採用 LOOK 演算法進行電梯調度，模擬真實電梯的運作情境。

---

## 模擬規則

| 項目 | 數值 |
|------|------|
| 樓層數 | 10 層 |
| 電梯數 | 2 部 |
| 每部電梯最大容量 | 5 人 |
| 每移動一層耗時 | 1 秒 |
| 每次停靠處理耗時 | 1 秒 |
| 每秒產生乘客數 | 1 人 |
| 模擬總人次 | 40 人 |

---

## 專案結構

```
ElevatorPractise/
├── main.go                   # 程式入口，生命週期 goroutine
├── config/
│   └── config.go             # 模擬參數常數定義
├── elevator/
│   ├── elevator.go           # 電梯結構與 LOOK 調度邏輯
│   └── elevatorsystem.go     # 電梯系統，管理電梯與候客佇列
└── passenger/
    ├── passenger.go          # 乘客結構與狀態定義
    └── passengermgr.go       # 乘客生成函式
```

---

## 運作流程

```
程式啟動
│
├─► 初始化 ElevatorSystem（建立 2 部電梯 + 10 層候客佇列）
│
├─► 啟動電梯 goroutine（go sys.Run()）
│     └─ 每秒 tick 一次，驅動電梯狀態機
│
└─► main goroutine 進入生命週期迴圈
      每秒：
        ① 從 ticker 接收 1 秒信號
        ② 隨機產生 1 位乘客（from ≠ to）
        ③ wg.Add(1)
        ④ 送入 PassengerCh（buffered channel）
      送完 40 人後 close(PassengerCh)
      執行 wg.Wait()，等待所有乘客抵達目的地
      印出總耗時，程式結束
```

### 電梯系統每秒 tick 流程

```
每秒 tick
│
├─ 1. 排乾 PassengerCh（drain loop）
│       取出所有已送入的乘客 → dispatch()
│
├─ 2. 兩部電梯各執行 step()
│       若當前樓層在目標集合中 → handleStop()（停靠 1 秒）
│       否則依 LOOK 演算法移動一層（移動 1 秒）
│
└─ 3. checkQueues()
        對尚未被指派電梯的候客樓層，重新嘗試指派
```

### 停靠處理順序（handleStop）

```
抵達目標樓層
├─ 1. 先下客：乘客 ToFloor == 當前樓層 → Status = Arrived，wg.Done()
├─ 2. 判斷下一段行進方向（nextDirection）
└─ 3. 再上客：從候客佇列接符合方向的乘客（直到滿員）
```

---

## 設計概念

### 並發架構

系統以兩個獨立的 goroutine 運行：

| Goroutine | 職責 |
|-----------|------|
| main goroutine | 每秒產生 1 位乘客，送入 channel |
| 電梯 goroutine（`sys.Run()`） | 每秒推進電梯狀態，處理乘客上下 |

兩者透過 **buffered channel**（`PassengerCh`）溝通，互不阻塞，各自維持獨立的 ticker 節奏。

### 電梯調度策略

收到新乘客時，依下列優先順序指派電梯：

1. **閒置電梯**：選距離最短的閒置電梯
2. **順路電梯**：選同方向、未滿員、且乘客樓層在前方的電梯（距離最短優先）
3. **等待**：以上皆不符合則加入樓層候客佇列，下一秒 `checkQueues()` 重試

### LOOK 演算法

電梯沿當前方向持續前進，停靠所有目標樓層，直到該方向無更多目標才反向。
相較於 SCAN 演算法，LOOK 不會跑到頂樓或底樓再折返，減少不必要的空跑。

```
目標：{3, 7, 9}，當前在 5 樓，方向 UP
→ 停靠 7、9
→ 方向切換為 DOWN
→ 停靠 3
```

---

## 物件說明

### `config` 套件

| 常數 | 說明 |
|------|------|
| `MaxFloor` | 大樓最高樓層（10） |
| `MaxCapacity` | 每部電梯最大載客數（5） |
| `TotalPassengers` | 模擬總乘客數（40） |
| `NumElevators` | 電梯數量（2） |
| `TickDuration` | 每個 tick 的時間長度（1 秒） |
| `PassengerChBuf` | PassengerCh 的 buffer 大小（10） |

---

### `passenger` 套件

#### `Passenger` 結構

| 欄位 | 型別 | 說明 |
|------|------|------|
| `ID` | int | 乘客編號 |
| `FromFloor` | int | 出發樓層 |
| `ToFloor` | int | 目標樓層 |
| `Status` | State | 當前狀態 |

#### `State` 乘客狀態機

```
Waiting → InElevator → Arrived
```

| 狀態 | 說明 |
|------|------|
| `Waiting` | 在樓層等待電梯 |
| `InElevator` | 已上電梯，前往目標樓層 |
| `Arrived` | 已抵達目標樓層 |

#### `NewPassenger(id int) *Passenger`

隨機產生乘客的出發與目標樓層（確保 `FromFloor ≠ ToFloor`），初始狀態為 `Waiting`。

---

### `elevator` 套件

#### `Elevator` 結構

| 欄位 | 型別 | 說明 |
|------|------|------|
| `ID` | int | 電梯編號 |
| `Floor` | int | 當前所在樓層 |
| `Dir` | Direction | 當前行進方向（IDLE / UP / DOWN） |
| `passengers` | []*Passenger | 電梯內的乘客清單 |
| `targets` | map[int]bool | 需要停靠的目標樓層集合 |

#### `Elevator` 方法

| 方法 | 說明 |
|------|------|
| `IsFull()` | 是否已達最大容量 |
| `IsIdle()` | 是否無目標且方向為 IDLE |
| `CanPickupEnRoute(floor, dir)` | 是否能在行進途中順路接乘客 |
| `StepsTo(floor)` | 距離某樓層的步數（絕對值） |
| `addTarget(floor)` | 新增目標樓層，並在 IDLE 時設定初始方向 |
| `step(sys)` | 每 tick 的狀態推進：停靠或移動一層 |
| `lookNextFloor()` | LOOK 演算法：決定下一步移動方向 |
| `handleStop(sys)` | 停靠處理：先下客、再依方向上客 |
| `nextDirection()` | 根據剩餘目標決定下一段行進方向 |

---

#### `ElevatorSystem` 結構

| 欄位 | 型別 | 說明 |
|------|------|------|
| `elevators` | []*Elevator | 所有電梯 |
| `floorQueues` | [][]*Passenger | 每層樓的候客佇列（index 0 = 1 樓） |
| `PassengerCh` | chan *Passenger | 接收新乘客的 buffered channel |
| `wg` | *sync.WaitGroup | 追蹤在途乘客數，歸零時結束模擬 |
| `mu` | sync.Mutex | 共享資料的 race condition 保護 |
| `tickCount` | int | 模擬時間計數（每 tick +1，用於日誌時戳） |

#### `ElevatorSystem` 方法

| 方法 | 說明 |
|------|------|
| `NewElevatorSystem(wg)` | 初始化系統，建立電梯與候客佇列 |
| `Run()` | 電梯 goroutine 主迴圈，每秒 tick 一次 |
| `dispatch(p)` | 將新乘客加入候客佇列並嘗試指派電梯 |
| `assignToElevator(p)` | 依優先策略為乘客選擇電梯 |
| `checkQueues()` | 檢查未被覆蓋的候客樓層，重新嘗試指派 |
| `allServed()` | 檢查是否所有乘客皆已抵達、電梯皆已清空 |

---

## 執行方式

```bash
go run .
```

### 範例輸出

```
Elevator 1 initialized at floor 1
Elevator 2 initialized at floor 1
Passenger  1 created: floor  1 → floor 10
[t=  1] Passenger  1 queued    at floor  1 →  10
[t=  1] Passenger  1 → Elevator 1 (idle,     dist=0)
[t=  1] Elevator 1 stopping at floor  1 (onboard=0)
[t=  1] Passenger  1 boarded   Elevator 1 at floor  1 →  10
[t=  2] Elevator 1 moved    → floor  2 (dir=UP, onboard=1)
...
[t= 67] All passengers served.
=== All 40 passengers served in 67.0 seconds ===
```
