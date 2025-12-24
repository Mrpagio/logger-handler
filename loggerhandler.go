
type MeterInterface interface {
	Int64Counter(name string, opts ...metric.InstrumentOption) (Int64CounterLike, error)
	Int64UpDownCounter(name string, opts ...metric.InstrumentOption) (Int64UpDownCounterLike, error)
}
