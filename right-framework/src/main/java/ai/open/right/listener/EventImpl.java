package ai.open.right.listener;

import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
@Slf4j
public class EventImpl implements Event {

    protected String workflow;

    protected String device;

    protected String type;

    protected Object data;

    protected String chat;

    protected String biz;

    protected Long now;

    public EventImpl(Event event) {
        this.workflow = event.getWorkflow();
        this.device = event.getDevice();
        this.chat = event.getChat();
        this.type = event.getType();
        this.data = event.getData();
        this.biz = event.getBiz();
        this.now = event.getNow();
    }

    public EventImpl() {
    }

    @Override
    public String getDimension() {
        return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
    }

    @Override
    public EventImpl init() {
        return this;
    }
}
