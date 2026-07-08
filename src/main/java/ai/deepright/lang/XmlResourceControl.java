package ai.deepright.lang;

import ai.open.right.protocol.ProtocolCode;

import ai.open.right.WorkflowException;

import java.io.BufferedInputStream;
import java.io.IOException;
import java.io.InputStream;
import java.net.URL;
import java.net.URLConnection;
import java.util.List;
import java.util.Locale;
import java.util.Properties;
import java.util.ResourceBundle;

public class XmlResourceControl extends ResourceBundle.Control {

    public static final String FORMAT_XML = "xml";

    @Override
    public List<String> getFormats(String baseName) {
        WorkflowException.check(baseName == null, "The base name must not be empty", ProtocolCode.C400);
        return List.of(XmlResourceControl.FORMAT_XML);
    }

    @Override
    public ResourceBundle newBundle(String baseName, Locale locale, String format, ClassLoader loader, boolean reload) throws IllegalAccessException, InstantiationException, IOException {
        WorkflowException.check(baseName == null, "The base name must not be empty", ProtocolCode.C400);
        WorkflowException.check(locale == null, "The locale must not be empty", ProtocolCode.C400);
        WorkflowException.check(format == null, "The format must not be empty", ProtocolCode.C400);
        WorkflowException.check(loader == null, "The loader must not be empty", ProtocolCode.C400);
        WorkflowException.check(!(XmlResourceControl.FORMAT_XML.equals(format)), "The format can not be support:" + format, ProtocolCode.C400);
        String resourceName = this.toResourceName(this.toBundleName(baseName, locale), XmlResourceControl.FORMAT_XML);
        ResourceBundle bundle = null;
        InputStream stream = null;
        if (reload) {
            URL url = loader.getResource(resourceName);
            if (url == null) {
                return null;
            }
            URLConnection connection = url.openConnection();
            connection.setUseCaches(false);
            stream = connection.getInputStream();
        } else {
            stream = loader.getResourceAsStream(resourceName);
        }
        if (stream == null) {
            return null;
        }
        try (InputStream input = new BufferedInputStream(stream)) {
            Properties properties = new Properties();
            properties.loadFromXML(input);
            bundle = new XmlResourceBundle(properties);
        }
        return bundle;
    }
}
