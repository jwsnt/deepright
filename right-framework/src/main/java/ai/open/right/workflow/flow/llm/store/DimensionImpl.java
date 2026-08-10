package ai.open.right.workflow.flow.llm.store;

import lombok.Getter;
import lombok.Setter;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
public class DimensionImpl implements Dimension {

    protected String workflow;

    protected String device;

    protected String chat;

    protected String biz;

    @Override
    public String getDimension() {
        return StringUtils.joinWith("-", this.getBiz(), this.getChat(), this.getDevice());
    }
}