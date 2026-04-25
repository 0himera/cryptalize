package domains

type DatetimeService struct{
	clock Clock
}

func (dt *DatetimeService) CurrentUnixSeconds() int64{
	return dt.clock.NowUnix()
}

func NewDatetimeService(clock Clock) *DatetimeService {
	return &DatetimeService{clock: clock}
}
