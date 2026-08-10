package ai.open.right.resouce;

import java.net.URL;

public interface ResourceService {

    public URL url(String location) throws Exception;

    public Class<?> root() throws Exception;
}
