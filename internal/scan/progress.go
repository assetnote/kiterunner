package scan

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/vbauerster/mpb/v6"
)

type ProgressBar struct {
	Pb       *mpb.Progress
	Requests *progressbar.ProgressBar
}

func NewProgress(max int64) *ProgressBar {
	pb := mpb.New(
		mpb.WithOutput(os.Stderr),
	)
	requestb := progressbar.NewOptions64(max,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWidth(5),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSetVisibility(true),
		progressbar.OptionSpinnerType(14),
		// progressbar.OptionFullWidth(),
	)
	return &ProgressBar{
		Pb:       pb,
		Requests: requestb,
	}
}

func (b *ProgressBar) Incr(n int64) {
	b.Requests.Add64(n)
}

func (b *ProgressBar) AddTotal(n int64) {
	b.Requests.ChangeMax64(b.Requests.GetMax64() + n)
}

