package ai.deepright.memory.knowledge;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.Getter;
import lombok.Setter;

@Getter
@Setter
public class Knowledge {

    protected Long lastUpdate;

    @JsonProperty("knowledge_disable")
    protected Boolean disable = false;

    protected String agentId;

    protected String path;

    public Boolean shouldUpdate(Long interval) throws Exception {
        // LastUpdate由调用端控制。首次为0，客户端会有防重间隔，超过间隔后会带为上次更新时间
        return this.lastUpdate == null || (System.currentTimeMillis() - this.lastUpdate) > interval;
    }
}
