package ai.open.right.workflow.flow;

import ai.open.right.context.RedirectContext;
import ai.open.right.context.UserContext;
import ai.open.right.workflow.flow.llm.store.Dimension;
import ai.open.right.workflow.flow.llm.store.history.History;
import ai.open.right.workflow.flow.media.MediaContext;
import ai.open.right.workflow.notify.NotifierWriteBack;

import java.util.List;
import java.util.Map;

public interface WorkflowTask extends NotifierWriteBack, RedirectContext, WorkflowObject, Dimension {

    public List<MediaContext> getMediaContext();

    public Map<String, Object> getMetadata();

    public UserContext getUserContext();

    public List<History> getHistories();

    public String getConversation();

    public String getNotifier();

    public String getProtocol();

    public String getWorkflow();

    public String getUpstream();

    // 获取间隔上次调用的耗时（用于性能检查）
    public Long getConsuming();

    public Long getCreated();

    public String getTrace();

    // 思考链（Workflow）当前Query（可被更改）
    public String getQuery();

    public String getChat();

    public String getBiz();

    public void setProviderAndToken(String provider, String token);

    public void setMediaContext(List<MediaContext> mediaContext);

    public void addMediaContext(MediaContext mediaContext);

    public void setUserContext(UserContext userContext);

    public void setHistories(List<History> histories);

    public void addHistories(List<History> histories);

    public void setWorkflow(String workflow);

    public void setNotifier(String notifier);

    public void setUpstream(String upstream);

    public void setProtocol(String protocol);

    public void setQuery(String query);

    public void setChat(String chat);

    public void setBiz(String biz);

    public <T> T getMetadata(String key, Class<T> clazz) throws Exception;

    public <T> T delMetadata(String key, Class<T> clazz) throws Exception;

    public void putMetadata(String key, Object val);

    public void delMetadata(String key);

    public Boolean containMetadata(String key);

    public Boolean containHistories();

    public WorkflowTask printQuery();

    // 置空Query
    public WorkflowTask emptyQuery();

    // 恢复最后Mark的Query
    public void resetQuery();

    // 恢复最后Mark的Query
    public void markQuery();
}
