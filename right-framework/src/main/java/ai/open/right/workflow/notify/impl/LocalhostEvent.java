package ai.open.right.workflow.notify.impl;

import ai.open.right.listener.Event;
import ai.open.right.workflow.flow.llm.Segment;
import com.fasterxml.jackson.annotation.JsonIgnore;
import org.apache.commons.lang3.StringUtils;

public class LocalhostEvent implements Event {

    public static final String TYPE = "localhost";

    @JsonIgnore
    protected final Segment body;

    @JsonIgnore
    protected final Long now;

    public LocalhostEvent(Segment segment) {
        this.body = segment;
        this.now = System.currentTimeMillis();
    }

    @Override
    public String getDimension() {
        return StringUtils.joinWith("-", this.body.getBiz(), this.body.getChat(), this.body.getDevice());
    }

    @Override
    public String getWorkflow() {
        return this.body.getWorkflow();
    }

    @Override
    public String getDevice() {
        return this.body.getUserContext().getDevice();
    }

    @Override
    public String getChat() {
        return this.body.getChat();
    }

    @Override
    public String getType() {
        return LocalhostEvent.TYPE;
    }

    @Override
    public Object getData() {
        return this.body;
    }

    @Override
    public String getBiz() {
        return this.body.getBiz();
    }

    @Override
    public Long getNow() {
        return this.now;
    }

    @Override
    public Event init() {
        return this;
    }
}
