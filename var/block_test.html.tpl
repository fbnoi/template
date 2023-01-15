{% extend "../var/template/base.html.tpl" %}
{% block content %}
<div>
    {% if show_content1 %}
        {{ show_content1 }}
    {% endif %}

    {% if show_content2 %}
        {{ content2 }}
    {% else %}
        <span>not show content2</span>
    {% endif %}

    {% for k, v in list %}
        {{ k }} : {{ v }}
    {% endfor %}
</div>
<hr>
{% include "../var/template/include_test.html.tpl" with PS(P("id", person)) %}
{% endblock %}