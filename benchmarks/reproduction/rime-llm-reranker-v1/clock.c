#include <lua.h>
#include <lauxlib.h>
#include <time.h>

static int milliseconds(lua_State *state) {
    struct timespec now;
    if (clock_gettime(CLOCK_MONOTONIC, &now) != 0) return luaL_error(state, "monotonic clock failed");
    lua_pushnumber(state, now.tv_sec * 1000.0 + now.tv_nsec / 1000000.0);
    return 1;
}

int luaopen_benchclock(lua_State *state) {
    lua_newtable(state);
    lua_pushcfunction(state, milliseconds);
    lua_setfield(state, -2, "milliseconds");
    return 1;
}
