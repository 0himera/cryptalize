package domains 

type Clock interface {
	NowUnix() int64
}
