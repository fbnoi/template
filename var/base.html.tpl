this is a line of texts
{{ some_content }}
{% block page %}
    some text in block page
    {% block content %}
    {% endblock %}
    some other text in block page
{% endblock %}
another line of texts