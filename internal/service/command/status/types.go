package status

import (
	"context"
	"io"
	"time"
)

const (
	SourcePackage = "package"
	SourceSystemd = "systemd"
)

type Options struct {
	ExcludeEnabled  bool
	IncludeDisabled bool
	OnlyPackage     bool
	OnlyClassic     bool
	IncludeTimeInfo bool
	SortAsc         bool
	SortDesc        bool
	Since           time.Duration
	NoMono          bool
	NoColor         bool
}

type Dependencies struct {
	Store   Store
	Out     io.Writer
	Now     func() time.Time
	MonoNow func() int64
}

type Store interface {
	List(ctx context.Context) ([]Service, error)
}

type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type Service struct {
	Name             string
	UnitName         string
	Source           string
	Enabled          bool
	Running          bool
	ActiveState      string
	SubState         string
	MainPID          int
	FragmentPath     string
	ActiveTime       time.Time
	ActiveTimeMono   int64
	InactiveTime     time.Time
	InactiveTimeMono int64
}

func (s Service) showTime() time.Time {
	if s.Running {
		return s.ActiveTime
	}
	return s.InactiveTime
}

func (s Service) showTimeMono() int64 {
	if s.Running {
		return s.ActiveTimeMono
	}
	return s.InactiveTimeMono
}
