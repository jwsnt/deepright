package ai.open.right.workflow.flow.track.impl;

import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;

public class RedisTrackSetCallback implements RedisCallback<Void> {

    protected final Integer expire;

    protected final byte[] keys;

    protected final byte[] data;

    public RedisTrackSetCallback(byte[] keys, byte[] data, Integer expire) {
        this.expire = expire;
        this.keys = keys;
        this.data = data;
    }

    @Override
    public Void doInRedis(RedisConnection connection) throws DataAccessException {
        connection.listCommands().rPush(this.keys, this.data);
        connection.keyCommands().expire(this.keys, this.expire);
        return null;
    }
}