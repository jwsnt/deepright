package ai.open.right.workflow.flow.llm.rag;

import ai.open.right.utils.JsonUtils;
import ai.open.right.utils.XmlUtils;
import ai.open.right.workflow.flow.llm.rag.future.RagFuture;
import org.apache.commons.lang3.StringUtils;

public interface RagService {

    public RagFuture rag(RagConfig ragConfig, RagData ragData) throws Exception;

    public static void updatePrompt(RagConfig ragConfig, RagData ragData, String key, Object val) throws Exception {
        RagService.updatePrompt(ragConfig, ragData, key, ragConfig.isMode(RagConfig.MODE_JSON) ? JsonUtils.write(val) : XmlUtils.write(val));
    }

    public static void updatePrompt(RagConfig ragConfig, RagData ragData, String key, String val) throws Exception {
        ragData.lock();
        try {
            val = StringUtils.defaultIfBlank(val, "");
            if (!StringUtils.isEmpty(key)) {
                String prompt = ragData.getPrompt();
                prompt = prompt.replace(key, val);
                ragData.setPrompt(prompt);
            } else {
                String query = ragData.getQuery().getQuery();
                if (!ragConfig.isOverride()) {
                    ragData.getQuery().setQuery(query + val);
                } else {
                    ragData.getQuery().setQuery(val);
                }
            }
        } finally {
            ragData.unlock();
        }
    }
}
