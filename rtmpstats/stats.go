// Package rtmpstats contains utilties for parsing, transforming, and
// aggregating data from an XML page exposed by nginx_rtmp_module's rtmp_stat
// directive.
package rtmpstats

import (
	"encoding/xml"
	"io"
	"time"
)

// Stats holds stats for the entirety of the nginx_rtmp_module.
type Stats struct {
	NGINXVersion     string        `xml:"nginx_version"`
	NGINXRTMPVersion string        `xml:"nginx_rtmp_version"`
	Compiler         string        `xml:"compiler"`
	Built            time.Time     `xml:"built"`
	PID              int           `xml:"pid"`
	Uptime           time.Duration `xml:"uptime"`
	Accepted         int           `xml:"naccepted"`
	BitrateIn        int           `xml:"bw_in"`
	BitrateOut       int           `xml:"bw_out"`
	BytesIn          int           `xml:"bytes_in"`
	BytesOut         int           `xml:"bytes_out"`
	Applications     []Application `xml:"server>application"`
}

func (s *Stats) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain Stats

	stats := struct {
		plain
		Built  Time     `xml:"built"`
		Uptime Duration `xml:"uptime"`
	}{}

	if err := d.DecodeElement(&stats, &start); err != nil {
		return err
	}

	*s = Stats(stats.plain)
	s.Built = time.Time(stats.Built)

	// The stats uptime is actually server uptime in seconds. Change to
	// seconds precision by dividing away the milliseconds and multipling
	// the seconds.
	s.Uptime = time.Duration(stats.Uptime) / time.Millisecond * time.Second
	return nil
}

// Application holds application-specific statistics.
type Application struct {
	Name    string   `xml:"name"`
	Streams []Stream `xml:"live>stream"`
}

// Stream holds stream-specific statistics.
type Stream struct {
	Name         string        `xml:"name"`
	Uptime       time.Duration `xml:"time"`
	BitrateIn    int           `xml:"bw_in"`
	BitrateOut   int           `xml:"bw_out"`
	BytesIn      int           `xml:"bytes_in"`
	BytesOut     int           `xml:"bytes_out"`
	BitrateVideo int           `xml:"bw_video"`
	BitrateAudio int           `xml:"bw_audio"`
	NumClients   int           `xml:"nclients"`
	Publishing   bool          `xml:"publishing"`
	Active       bool          `xml:"active"`

	// Meta information on Video
	VideoWidth     int     `xml:"meta>video>width"`
	VideoHeight    int     `xml:"meta>video>height"`
	VideoFramerate int     `xml:"meta>video>frame_rate"`
	VideoCodec     string  `xml:"meta>video>codec"`
	VideoProfile   string  `xml:"meta>video>profile"`
	VideoCompat    int     `xml:"meta>video>compat"`
	VideoLevel     float64 `xml:"meta>video>level"`

	// Meta information on Audio
	AudioCodec      string `xml:"meta>audio>codec"`
	AudioProfile    string `xml:"meta>audio>profile"`
	AudioChannels   int    `xml:"meta>audio>channels"`
	AudioSampleRate int    `xml:"meta>audio>sample_rate"`

	Clients []Client `xml:"client"`
}

func (s *Stream) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain Stream

	stats := struct {
		plain
		Uptime     Duration `xml:"time"`
		Publishing Boolean  `xml:"publishing"`
		Active     Boolean  `xml:"active"`
	}{}

	if err := d.DecodeElement(&stats, &start); err != nil {
		return err
	}

	*s = Stream(stats.plain)
	s.Uptime = time.Duration(stats.Uptime)
	s.Publishing = bool(stats.Publishing)
	s.Active = bool(stats.Active)
	return nil
}

// Client holds client-specific statistics.
type Client struct {
	ID            int           `xml:"id"`
	Address       string        `xml:"address"`
	Uptime        time.Duration `xml:"time"`
	FlashVersion  string        `xml:"flashver"`
	PageURL       string        `xml:"pageurl"`
	SWFURL        string        `xml:"swfurl"`
	DroppedFrames int           `xml:"dropped"`
	AVSync        int           `xml:"avsync"`
	Timestamp     time.Duration `xml:"timestamp"`
	Active        bool          `xml:"active"`
	Publishing    bool          `xml:"publishing"`
}

func (c *Client) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain Client

	stats := struct {
		plain
		Uptime     Duration `xml:"time"`
		Timestamp  Duration `xml:"timestamp"`
		Active     Boolean  `xml:"active"`
		Publishing Boolean  `xml:"publishing"`
	}{}

	if err := d.DecodeElement(&stats, &start); err != nil {
		return err
	}

	*c = Client(stats.plain)
	c.Uptime = time.Duration(stats.Uptime)
	c.Timestamp = time.Duration(stats.Timestamp)
	c.Active = bool(stats.Active)
	c.Publishing = bool(stats.Publishing)
	return nil
}

func Unmarshal(r io.Reader) (*Stats, error) {
	var (
		s   Stats
		err error
	)

	dec := xml.NewDecoder(r)

	err = dec.Decode(&s)
	return &s, err
}
