package ai.deepright.llm.token;

import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.token.TokenData;

public interface TokenNotifier {

    public void notify(ProviderRequest request, TokenData tokenData) throws Exception;
}
