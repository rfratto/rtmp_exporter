package rtmpstats

import (
	"encoding/xml"
	"time"
)

// nginx_rtmp_module uses some non-Go values for its fields but we want to expose
// them as native objects. This file contains all the types for the non-Go values
// we want to parse but load them into native Go types.
//
// None of these types will be exposed directly by our stats struct. Rather any
// struct that needs access to one of these types will override UnmarshalXML, embed
// the final type, but override fields with one of these types. See Stats.UnmarshalXML
// for an example of this in action.

// Time is a time.Time that unmarshals correctly using nginx_rtmp_module's time format.
type Time time.Time

func (t *Time) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var timeStr string
	if err := d.DecodeElement(&timeStr, &start); err != nil {
		return err
	}

	parsedTime, err := time.Parse("Jan _2 2006 15:04:05", timeStr)
	if err != nil {
		return err
	}

	*t = Time(parsedTime)
	return nil
}

// Duration is a time.Duration that unmarshals correctly using
// nginx_rtmp_module's duration format (milliseconds).
type Duration time.Duration

func (d *Duration) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	var ms int
	if err := dec.DecodeElement(&ms, &start); err != nil {
		return err
	}

	*d = Duration(time.Millisecond * time.Duration(ms))
	return nil
}

// Boolean is a bool that is true if UnmarshalXML is called. It's intended
// for self-closing tags where their presence indicate truthiness.
type Boolean bool

func (b *Boolean) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	*b = true

	// We still need to consume the element so we'll pretend to parse
	// it here.
	var consume bool
	err := dec.DecodeElement(&dec, &start)
	_ = consume

	return err
}
