// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT

// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-ignore: Unused imports
import { Create as $Create } from "@wailsio/runtime";

/**
 * A Time represents an instant in time with nanosecond precision.
 * 
 * Programs using times should typically store and pass them as values,
 * not pointers. That is, time variables and struct fields should be of
 * type [time.Time], not *time.Time.
 * 
 * A Time value can be used by multiple goroutines simultaneously except
 * that the methods [Time.GobDecode], [Time.UnmarshalBinary], [Time.UnmarshalJSON] and
 * [Time.UnmarshalText] are not concurrency-safe.
 * 
 * Time instants can be compared using the [Time.Before], [Time.After], and [Time.Equal] methods.
 * The [Time.Sub] method subtracts two instants, producing a [Duration].
 * The [Time.Add] method adds a Time and a Duration, producing a Time.
 * 
 * The zero value of type Time is January 1, year 1, 00:00:00.000000000 UTC.
 * As this time is unlikely to come up in practice, the [Time.IsZero] method gives
 * a simple way of detecting a time that has not been initialized explicitly.
 * 
 * Each time has an associated [Location]. The methods [Time.Local], [Time.UTC], and Time.In return a
 * Time with a specific Location. Changing the Location of a Time value with
 * these methods does not change the actual instant it represents, only the time
 * zone in which to interpret it.
 * 
 * Representations of a Time value saved by the [Time.GobEncode], [Time.MarshalBinary], [Time.AppendBinary],
 * [Time.MarshalJSON], [Time.MarshalText] and [Time.AppendText] methods store the [Time.Location]'s offset,
 * but not the location name. They therefore lose information about Daylight Saving Time.
 * 
 * In addition to the required “wall clock” reading, a Time may contain an optional
 * reading of the current process's monotonic clock, to provide additional precision
 * for comparison or subtraction.
 * See the “Monotonic Clocks” section in the package documentation for details.
 * 
 * Note that the Go == operator compares not just the time instant but also the
 * Location and the monotonic clock reading. Therefore, Time values should not
 * be used as map or database keys without first guaranteeing that the
 * identical Location has been set for all values, which can be achieved
 * through use of the UTC or Local method, and that the monotonic clock reading
 * has been stripped by setting t = t.Round(0). In general, prefer t.Equal(u)
 * to t == u, since t.Equal uses the most accurate comparison available and
 * correctly handles the case when only one of its arguments has a monotonic
 * clock reading.
 */
export type Time = any;
