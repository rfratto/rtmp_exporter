package rtmpstats

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gotest.tools/assert"
)

func TestWithStreamMapper(t *testing.T) {
	t.Run("duplicates", func(t *testing.T) {
		input := &Stats{
			Applications: []Application{{
				Streams: []Stream{
					{Name: "stream_a"},
					{Name: "stream_b"},
				},
			}},
		}

		mapper := func(_ string) string { return "whoops" }
		err := WithStreamMapper(mapper)(input)
		require.EqualError(t, err, "a stream with the name whoops already exists")
	})

	t.Run("rename", func(t *testing.T) {
		input := &Stats{
			Applications: []Application{{
				Streams: []Stream{
					{Name: "stream_a"},
					{Name: "stream_b"},
				},
			}},
		}

		uppercase := func(s string) string { return strings.ToUpper(s) }
		err := WithStreamMapper(uppercase)(input)
		require.NoError(t, err)

		expect := []Stream{
			{Name: "STREAM_A"},
			{Name: "STREAM_B"},
		}
		require.Equal(t, expect, input.Applications[0].Streams)
	})
}

func TestWithClientMapper(t *testing.T) {
	input := &Stats{
		Applications: []Application{{
			Streams: []Stream{{
				Name: "stream",
				Clients: []Client{
					{
						ID:            "sum_a",
						Address:       "127.0.0.1",
						Uptime:        time.Minute,
						FlashVersion:  "1",
						PageURL:       "http://localhost/watch",
						SWFURL:        "https://swf",
						DroppedFrames: 150,
						AVSync:        -12,
						Timestamp:     time.Minute,
						Active:        false,
						Publishing:    false,
					},
					{
						ID:            "other_a",
						Address:       "1.1.1.1",
						Uptime:        time.Minute,
						FlashVersion:  "2",
						PageURL:       "http://other.localhost/watch",
						SWFURL:        "https://swf2",
						DroppedFrames: 0,
						AVSync:        0,
						Timestamp:     time.Minute,
						Active:        false,
						Publishing:    false,
					},
					{
						ID:            "sum_b",
						Address:       "127.0.0.2",
						Uptime:        time.Second,
						FlashVersion:  "1.1",
						PageURL:       "http://localhost/watch-also",
						SWFURL:        "https://swf/1",
						DroppedFrames: 100,
						AVSync:        -12,
						Timestamp:     time.Second,
						Active:        true,
						Publishing:    false,
					},
				},
			}},
		}},
	}

	combineSums := func(stream string, in string) string {
		assert.Equal(t, "stream", stream)
		if in == "sum_a" || in == "sum_b" {
			return "sum"
		}
		return in
	}
	err := WithClientMapper(combineSums)(input)
	require.NoError(t, err)

	expect := []Client{
		{
			ID:            "sum",
			Address:       "127.0.0.1",
			Uptime:        time.Minute,
			FlashVersion:  "1",
			PageURL:       "http://localhost/watch",
			SWFURL:        "https://swf",
			DroppedFrames: 250,
			AVSync:        -12,
			Timestamp:     time.Minute,
			Active:        true,
			Publishing:    false,
		},
		{
			ID:            "other_a",
			Address:       "1.1.1.1",
			Uptime:        time.Minute,
			FlashVersion:  "2",
			PageURL:       "http://other.localhost/watch",
			SWFURL:        "https://swf2",
			DroppedFrames: 0,
			AVSync:        0,
			Timestamp:     time.Minute,
			Active:        false,
			Publishing:    false,
		},
	}
	require.Equal(t, expect, input.Applications[0].Streams[0].Clients)
}
