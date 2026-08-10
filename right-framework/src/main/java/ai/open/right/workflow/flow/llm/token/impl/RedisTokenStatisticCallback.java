package ai.open.right.workflow.flow.llm.token.impl;

import ai.open.right.workflow.flow.llm.token.TokenData;
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;

public class RedisTokenStatisticCallback implements RedisCallback<Void> {

    protected final TokenData tokenData;

    protected final byte[] key4thinking;

    protected final byte[] key4input;

    protected final byte[] key4token;

    protected final byte[] key4cache;

    protected final Integer expire;

    public RedisTokenStatisticCallback(byte[] key4thinking, byte[] key4input, byte[] key4token, byte[] key4cache, TokenData tokenData, Integer expire) {
        this.key4thinking = key4thinking;
        this.key4input = key4input;
        this.key4token = key4token;
        this.key4cache = key4cache;
        this.tokenData = tokenData;
        this.expire = expire;
    }

    @Override
    public Void doInRedis(RedisConnection connection) throws DataAccessException {
        connection.stringCommands().incrBy(this.key4thinking, this.tokenData.getThinking());
        connection.stringCommands().incrBy(this.key4input, this.tokenData.getInput());
        connection.stringCommands().incrBy(this.key4cache, this.tokenData.getCache());
        connection.stringCommands().incrBy(this.key4token, this.tokenData.getTotal());
        connection.keyCommands().expire(this.key4thinking, this.expire);
        connection.keyCommands().expire(this.key4input, this.expire);
        connection.keyCommands().expire(this.key4cache, this.expire);
        connection.keyCommands().expire(this.key4token, this.expire);
        return null;
    }
}