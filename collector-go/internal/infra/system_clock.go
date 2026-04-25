package infra

import "time"

type SystemClock struct{}

func (SystemClock) NowUnix() int64{
	return time.Now().Unix()
}
