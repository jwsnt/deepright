package ai.open.right.workflow.flow.llm.provider;

import java.util.Map;

public interface ProviderImageConfig {

    public void setImageConfig(Map<String, Object> imageConfig);

    public Map<String, Object> getImageConfig();
}
