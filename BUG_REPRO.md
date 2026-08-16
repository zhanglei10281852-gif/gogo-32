# Bug Reproduction

## 包的性质

当前 test_model_fix 保存的是被测模型修复后的结果源码，不是初始含 Bug 源码。要复现原始缺陷，必须检出下面固定的 parent SHA；不要在当前修复结果源码上期待重新出现修复前失败。生成系统使用的可信验证补丁和完整验证日志仅在本地留存，不提交到结果分支。

## 问题现象

两个时段的合法进口价都为 9223372036854775807、出口价仅为 1 时，启用 export 的计划却产生了 V2G 放电动作；如此低的出口价不应被判定为有吸引力。请修复费率统计与导出决策，保持其他排序和统计行为不变，并保证全量测试通过。

## 含 Bug 版本

- 仓库：zhanglei10281852-gif/gogo-32
- 仓库地址：https://github.com/zhanglei10281852-gif/gogo-32.git
- parent SHA：b1595c5f898ea44df748174f8049a7d87b702284

## 复现步骤

```bash
git clone -- https://github.com/zhanglei10281852-gif/gogo-32.git bug-repro
cd bug-repro
git checkout --detach b1595c5f898ea44df748174f8049a7d87b702284
go test ./dispatch -run "^TestHugeImportPricesDoNotMakeCheapExportAttractive$" -count=1 -v
```

## 双架构完整错误信息

### linux/amd64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./dispatch -run "^TestHugeImportPricesDoNotMakeCheapExportAttractive$" -count=1 -v
=== RUN   TestHugeImportPricesDoNotMakeCheapExportAttractive
    tariff_overflow_test.go:42: unexpected V2G action for negligible export price: model.Action{IntervalIndex:0, VehicleID:"ev", PowerW:-100, GridEnergyWh:100, BatteryDeltaWh:-100, SOCBeforeWh:900, SOCAfterWh:800, Reason:"price-responsive constrained export"}
--- FAIL: TestHugeImportPricesDoNotMakeCheapExportAttractive (0.00s)
FAIL
FAIL	gridflex/dispatch	0.003s
FAIL

```

stderr：

```text
(empty)
```

### linux/arm64

- 容器内复现预期退出码：1
- 容器内复现实际退出码：1

stdout：

```text
$ go test ./dispatch -run "^TestHugeImportPricesDoNotMakeCheapExportAttractive$" -count=1 -v
=== RUN   TestHugeImportPricesDoNotMakeCheapExportAttractive
    tariff_overflow_test.go:42: unexpected V2G action for negligible export price: model.Action{IntervalIndex:0, VehicleID:"ev", PowerW:-100, GridEnergyWh:100, BatteryDeltaWh:-100, SOCBeforeWh:900, SOCAfterWh:800, Reason:"price-responsive constrained export"}
--- FAIL: TestHugeImportPricesDoNotMakeCheapExportAttractive (0.01s)
FAIL
FAIL	gridflex/dispatch	0.136s
FAIL

```

stderr：

```text
(empty)
```

## 通过条件

极大非负费率的平均值计算正确；出口价 1 不触发 V2G；Statistics 的各平均字段同样保持正确；双架构定向/全量/build/vet 通过。
