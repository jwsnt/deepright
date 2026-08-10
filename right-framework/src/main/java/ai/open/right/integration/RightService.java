package ai.open.right.integration;

import java.util.concurrent.Future;

public interface RightService {

    public Future<String> get(RightConfig rightConfig) throws Exception;
}
