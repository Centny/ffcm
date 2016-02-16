package ffcm

import (
	"bufio"
	"github.com/Centny/gwf/log"
	"github.com/Centny/gwf/netw/dtm"
	"github.com/Centny/gwf/routing"
	"github.com/Centny/gwf/util"
	"net/http"
	"strings"
)

var DTCM_C *dtm.DTM_C

func RunFFCM_C(cfg string) error {
	if DTCM_C != nil {
		return util.Err("runner is running")
	}
	var fcfg = util.NewFcfg3()
	fcfg.InitWithFilePath2(cfg, true)
	fcfg.Print()
	var HTTP = &http.Server{}
	HTTP.Addr = fcfg.Val("listen")
	routing.HFunc("^/notify(\\?.*)?$", NofityProc)
	HTTP.Handler = routing.Shared
	routing.Shared.Print()
	routing.Shared.ShowLog = true
	DTCM_C = dtm.StartDTM_C(fcfg)
	log.D("listen web on %v", HTTP.Addr)
	return HTTP.ListenAndServe()
}

/*
frame=497
fps=50.5
stream_0_0_q=-1.0
bitrate=1812.2kbits/s
total_size=4783970
out_time_ms=21119167
out_time=00:00:21.119167
dup_frames=0
drop_frames=0
progress=end
*/
type Progress struct {
	Frame      int     `m2s:"frame" json:"frame"`
	FPS        float64 `m2s:"fps" json:"fps"`
	Stream     float64 `m2s:"stream" json:"stream"`
	Bitrate    string  `m2s:"bitrate" json:"bitrate"`
	TotalSize  int64   `m2s:"bitrate" json:"bitrate"`
	OutTimeMs  int64   `m2s:"out_time_ms" json:"out_time_ms"`
	OutTime    string  `m2s:"out_time" json:"out_time"`
	DupFrames  int     `m2s:"dup_frames" json:"dup_frames"`
	DropFrames int     `m2s:"drop_frames" json:"drop_frames"`
	Progress   string  `m2s:"progress" json:"progress"`
}

func NofityProc(hs *routing.HTTPSession) routing.HResult {
	if DTCM_C == nil {
		return hs.Printf("runner is not running")
	}
	var tid string
	var duration int64
	err := hs.ValidCheckVal(`
		tid,R|S,L:0;
		duration,R|I,R:0;
		`, &tid, &duration)
	if err != nil {
		return hs.Printf("valid argument error by %v", err)
	}
	var reader = bufio.NewReader(hs.R.Body)
	var frame = util.Map{}
	for {
		bys, err := util.ReadLine(reader, 102400, false)
		if err != nil {
			break
		}
		line := strings.Trim(string(bys), " \n\t")
		lines := strings.SplitN(line, "=", 2)
		lines[0] = strings.Trim(lines[0], " \t")
		if len(lines) < 2 {
			frame[lines[0]] = ""
		} else {
			frame[lines[0]] = lines[1]
		}
		if lines[0] != "progress" {
			continue
		}
		var progress Progress
		frame.ToS(&progress)
		var rate = float64(progress.OutTimeMs) / float64(duration)
		if int(rate*1000)%10 == 0 {
			log.D("NofityProc receive rate(%v%%) to task(%v),duration(%v)", int(rate*100), tid, duration)
		}
		err = DTCM_C.NotifyProc(tid, rate)
		if err != nil {
			return hs.Printf("notify procgress to task(%v) error by %v", tid, err)
		}
		frame = util.Map{}
	}
	return hs.Printf("%v", "DONE")
}
