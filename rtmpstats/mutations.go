package rtmpstats

import "fmt"

// It's common for RTMP servers to use special keys for pushing to a stream,
// but operators might not want to expose those keys as labels in the exporter.
// This file provides utilities for remapping values.
//
// If the result of mapping values results in two objects with the same key,
// they will automatically be aggregated (summed) together and the count of
// objects used for that sum will be stored in a EntriesCount field.

// Mutator is any function that mutates Stats.
type Mutator func(s *Stats) error

// WithStreamMapper creates a Mutator that mutates a Stats, changing all stream
// names with the result of the mapper function. Resulting streams must have unique
// names. The mutator will fail if names are not unique post-mapping.
func WithStreamMapper(mapper func(in string) string) Mutator {
	return func(s *Stats) error {
		for i, app := range s.Applications {
			// Transformed set of streams. We don't transform in-place so an invalid
			// mapping doesn't partially mutate the set.
			transformed := make([]Stream, 0, len(app.Streams))
			streamLookup := make(map[string]struct{})

			for _, stream := range app.Streams {
				stream.Name = mapper(stream.Name)

				if _, found := streamLookup[stream.Name]; found {
					return fmt.Errorf("a stream with the name %s already exists", stream.Name)
				}
				streamLookup[stream.Name] = struct{}{}
				transformed = append(transformed, stream)
			}

			s.Applications[i].Streams = transformed
		}

		return nil
	}
}

// WithClientMapper creates a Mutator that mutates a Stats, changing all client
// names with the result of the mapper function. Mapper will be invoked with the
// stream the client is in along with the input client ID. Resulting clients
// that have the same ID will be aggregated together. The EntriesCount field on the
// client will hold the final result of entries that were aggregated together (1 if
// no aggregation was performed).
func WithClientMapper(mapper func(stream string, in string) string) Mutator {
	return func(s *Stats) error {
		for appIdx, app := range s.Applications {
			for streamIdx, stream := range app.Streams {
				aggregated := make([]Client, 0, len(stream.Clients))
				clientLookup := make(map[string]int)

				for _, client := range stream.Clients {
					client.ID = mapper(stream.Name, client.ID)

					// If the client already exists in the map, two clients resulted
					// in the same mapping and we need to aggregate them together
					// now.
					duplicateIdx, found := clientLookup[client.ID]
					if !found {
						clientLookup[client.ID] = len(aggregated)
						aggregated = append(aggregated, client)
						continue
					}

					aggregated[duplicateIdx] = aggregated[duplicateIdx].Add(client)
				}

				s.Applications[appIdx].Streams[streamIdx].Clients = aggregated
			}
		}

		return nil
	}
}
