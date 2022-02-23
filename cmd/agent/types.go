package main

type gauge float64
type counter int64

type Payload struct {
	Alloc,
	BuckHashSys,
	Frees,
	GCCPUFraction,
	GCSys,
	HeapAlloc,
	HeapIdle,
	HeapInuse,
	HeapObjects,
	HeapReleased,
	HeapSys,
	LastGC,
	Lookups,
	MCacheInuse,
	MCacheSys,
	MSpanInuse,
	MSpanSys,
	Mallocs,
	NextGC,
	NumForcedGC,
	NumGC,
	OtherSys,
	PauseTotalNs,
	StackInuse,
	StackSys,
	Sys,
	RandomValue gauge // — обновляемое рандомное значение.

	PollCount counter // — счётчик, увеличивающийся на 1 при каждом обновлении метрики из пакета runtime (на каждый pollInterval — см. ниже).
}
