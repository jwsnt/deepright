package ai.open.right.workflow.flow.command.impl;

import ai.open.right.WorkflowException;
import ai.open.right.utils.GzipUtils;
import ai.open.right.utils.JsonUtils;
import ai.open.right.workflow.flow.command.QuickCommand;
import lombok.extern.slf4j.Slf4j;
import org.springframework.dao.DataAccessException;
import org.springframework.data.redis.connection.RedisConnection;
import org.springframework.data.redis.core.RedisCallback;

import java.util.List;

@Slf4j
public class RedisCallbackImpl implements RedisCallback<Void> {

    protected final List<QuickCommand> commands;

    protected final Integer expire;

    protected final byte[] kBytes;

    public RedisCallbackImpl(List<QuickCommand> commands, Integer expire, byte[] kBytes) {
        this.commands = commands;
        this.expire = expire;
        this.kBytes = kBytes;
    }

    @Override
    public Void doInRedis(RedisConnection connection) throws DataAccessException {
        // Remove all，每次更新前删除同纬度所有其他
        connection.zSetCommands().zRemRange(this.kBytes, 0, -1);
        for (QuickCommand command : this.commands) {
            try {
                // 追加优先级
                connection.zAdd(this.kBytes, command.getPriority(), GzipUtils.compress(JsonUtils.write(command)));
                // 设置过期时间(SECONDS)/刷新
                connection.keyCommands().expire(this.kBytes, this.expire);
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
        return null;
    }
}