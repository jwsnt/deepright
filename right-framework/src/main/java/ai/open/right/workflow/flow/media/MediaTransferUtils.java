package ai.open.right.workflow.flow.media;

import ai.open.right.WorkflowException;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;

import java.net.URI;
import java.net.URISyntaxException;
import java.util.Set;

@Slf4j
public class MediaTransferUtils {

    public static final Set<String> URI = Set.of("https", "http", "ftps", "ftp");

    public static Boolean isNetwork(String uri) throws Exception {
        try {
            String scheme = new URI(uri).getScheme();
            return scheme != null && MediaTransferUtils.URI.contains(StringUtils.lowerCase(scheme));
        } catch (URISyntaxException e) {
            if (log.isDebugEnabled()) {
                log.debug(e.getMessage(), e);
            }
            return false;
        } catch (Exception e) {
            WorkflowException.dolog(e);
            return false;
        }
    }
}
