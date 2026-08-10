package ai.open.right.workflow.flow.pubsub;

import lombok.Getter;
import lombok.Setter;
import lombok.ToString;
import org.apache.commons.lang3.StringUtils;

@Setter
@Getter
@ToString
public class PubSubQuery {

    protected String answer;

    protected String query;

    protected String key;

    public Boolean hasAnswer() {
        return !StringUtils.isEmpty(this.answer);
    }
}
