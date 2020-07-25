package rtmpstats

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnmarshal(t *testing.T) {
	f, err := os.Open("testdata/stats.xml")
	require.NoError(t, err)
	defer f.Close()

	s, err := Unmarshal(f)
	require.NoError(t, err)

	expect := &Stats{
		NGINXVersion:     "1.19.0",
		NGINXRTMPVersion: "1.1.4",
		Compiler:         "gcc 9.3.0 (Alpine 9.3.0) ",
		Built:            time.Date(2020, time.July, 11, 22, 3, 37, 0, time.UTC),
		PID:              13,
		Uptime:           93879 * time.Second,
		Accepted:         11,

		BitrateIn:  2338696,
		BitrateOut: 7016072,
		BytesIn:    130057972,
		BytesOut:   239470507,

		Applications: []Application{{
			Name: "live",
			Streams: []Stream{{
				Name:         "streamName",
				Uptime:       500003 * time.Millisecond,
				BitrateIn:    2333128,
				BitrateOut:   6999400,
				BytesIn:      129733847,
				BytesOut:     238916032,
				BitrateVideo: 2226200,
				BitrateAudio: 106920,
				NumClients:   4,
				Publishing:   true,
				Active:       true,

				VideoWidth:     1920,
				VideoHeight:    1080,
				VideoFramerate: 30,
				VideoCodec:     "H264",
				VideoProfile:   "High",
				VideoCompat:    0,
				VideoLevel:     4,

				AudioCodec:      "AAC",
				AudioProfile:    "LC",
				AudioChannels:   2,
				AudioSampleRate: 44100,

				Clients: []Client{
					{
						ID:            "51",
						Address:       "1.1.1.51",
						Uptime:        time.Millisecond * 36310,
						FlashVersion:  "WIN 32,0,0,403",
						PageURL:       "http://localhost/watch",
						SWFURL:        "https://vjs.zencdn.net/swf/5.4.2/video-js.swf",
						DroppedFrames: 0,
						AVSync:        -12,
						Timestamp:     time.Millisecond * 499599,
						Active:        true,
						EntriesCount:  1,
					},
					{
						ID:            "15",
						Address:       "1.1.1.15",
						Uptime:        time.Millisecond * 371856,
						FlashVersion:  "MAC 32,0,0,403",
						PageURL:       "http://localhost/watch",
						SWFURL:        "https://vjs.zencdn.net/swf/5.4.2/video-js.swf",
						DroppedFrames: 0,
						AVSync:        -12,
						Timestamp:     time.Millisecond * 499599,
						Active:        true,
						EntriesCount:  1,
					},
					{
						ID:            "3",
						Address:       "127.0.0.1",
						Uptime:        time.Millisecond * 496931,
						FlashVersion:  "LNX 9,0,124,2",
						DroppedFrames: 0,
						AVSync:        -12,
						Timestamp:     time.Millisecond * 499599,
						Active:        true,
						EntriesCount:  1,
					},
					{
						ID:            "1",
						Address:       "1.1.1.1",
						Uptime:        time.Millisecond * 500278,
						FlashVersion:  "FMLE/3.0 (compatible; FMSc/1.0)",
						SWFURL:        "rtmp://localhost/live",
						DroppedFrames: 0,
						AVSync:        -12,
						Timestamp:     time.Millisecond * 499599,
						Publishing:    true,
						Active:        true,
						EntriesCount:  1,
					},
				},
			}},
		}},
	}
	require.Equal(t, expect, s)
}
