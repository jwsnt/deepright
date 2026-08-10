package ai.open.right.workflow.flow.llm.provider;

import java.util.List;
import java.util.Map;

// Fun Call
public interface ProviderFunCall {

    public void setName(String name);

    public void setDescription(String description);

    public void setRequired(List<String> required);

    public void setProperties(Map<String, Object> properties);

    public Map<String, Object> getProperties();

    public List<String> getRequired();

    public String getDescription();

    public String getName();
}
