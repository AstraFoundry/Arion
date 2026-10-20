#include <mach/mach_time.h>
#include <pthread/qos.h>
#include <stdint.h>
#include <stdio.h>
#include <time.h>
static double posix_ms(clockid_t id) { struct timespec ts; clock_gettime(id, &ts); return ts.tv_sec * 1000.0 + ts.tv_nsec / 1000000.0; }
static double mach_ms(uint64_t tick, mach_timebase_info_data_t base) { return (double)tick * base.numer / base.denom / 1000000.0; }
int main(void) {
 mach_timebase_info_data_t base; mach_timebase_info(&base);
 double mono = posix_ms(CLOCK_MONOTONIC), uptime = posix_ms(CLOCK_UPTIME_RAW);
 double absolute = mach_ms(mach_absolute_time(), base), continuous = mach_ms(mach_continuous_time(), base);
 struct timespec wait = {0, 250000000}; nanosleep(&wait, NULL);
 printf("{\"architecture\":\"arm64\",\"qosClass\":%u,\"timebaseNumer\":%u,\"timebaseDenom\":%u,\"monotonicStartMs\":%.6f,\"uptimeStartMs\":%.6f,\"machAbsoluteStartMs\":%.6f,\"continuousStartMs\":%.6f,\"monotonicElapsedMs\":%.6f,\"uptimeElapsedMs\":%.6f,\"absoluteElapsedMs\":%.6f,\"continuousElapsedMs\":%.6f}\n",(unsigned)qos_class_self(),base.numer,base.denom,mono,uptime,absolute,continuous,posix_ms(CLOCK_MONOTONIC)-mono,posix_ms(CLOCK_UPTIME_RAW)-uptime,mach_ms(mach_absolute_time(),base)-absolute,mach_ms(mach_continuous_time(),base)-continuous);
 return 0;
}
