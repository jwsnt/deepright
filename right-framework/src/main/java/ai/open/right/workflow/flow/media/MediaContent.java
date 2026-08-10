package ai.open.right.workflow.flow.media;

import lombok.Getter;
import lombok.Setter;
import org.apache.commons.collections.CollectionUtils;
import org.apache.commons.collections.MapUtils;
import org.apache.commons.lang3.StringUtils;

import java.util.List;
import java.util.Map;

@Setter
@Getter
public class MediaContent {

    protected List<MediaContext> mediaContext;

    protected Map<String, Object> metadata;

    protected String query;

    public Boolean hasMediaContext() {
        return !CollectionUtils.isEmpty(this.mediaContext);
    }

    public Boolean hasMetadata() {
        return !MapUtils.isEmpty(this.metadata);
    }

    public Boolean hasQuery() {
        return !StringUtils.isEmpty(this.query);
    }
}
