package ai.open.right.workflow.flow.llm.provider;

import ai.open.right.listener.Event;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;
import org.apache.commons.lang3.StringUtils;

@Getter
public class ProviderEvent implements Event {

    public static final String TYPE = "provider";

    @JsonIgnore
    // LLM配置
    protected final ProviderRequest providerRequest;

    @JsonIgnore
    // LLM请求/响应
    protected final ProviderData providerData;

    public ProviderEvent(ProviderRequest providerRequest) {
        this.providerData = (this.providerRequest = providerRequest).getProviderData();
    }

    @Override
    public String getDimension() {
        return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
    }

    @Override
    public String getWorkflow() {
        return this.providerRequest.getMessage().getWorkflow();
    }

    @Override
    public String getDevice() {
        return this.providerRequest.getMessage().getDevice();
    }

    @Override
    public Object getData() {
        return this.providerData;
    }

    @Override
    public String getChat() {
        return this.providerRequest.getMessage().getChat();
    }

    @Override
    public String getType() {
        return ProviderEvent.TYPE;
    }


    @Override
    public String getBiz() {
        return this.providerRequest.getMessage().getBiz();
    }

    @Override
    public Long getNow() {
        return this.providerRequest.getMessage().getCreated();
    }

    @Override
    public Event init() {
        return this;
    }
}
