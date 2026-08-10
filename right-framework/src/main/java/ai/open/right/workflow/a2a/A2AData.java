package ai.open.right.workflow.a2a;

import ai.open.right.workflow.flow.llm.Segment;
import com.fasterxml.jackson.annotation.JsonIgnore;
import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.LinkedHashMap;

@Setter
@Getter
public class A2AData extends LinkedHashMap<String, Object> implements A2AProtocol {

    @JsonIgnore
    protected Segment segment;

    public A2AData bindSegment(Segment segment) {
        this.segment = segment;
        return this;
    }

    public Boolean isSupport(String internal) {
        return StringUtils.equalsIgnoreCase(MapUtils.getString(this, "internal"), internal);
    }

    @Override
    public A2AData reset() {
        this.put("internal", null);
        return this;
    }
}
