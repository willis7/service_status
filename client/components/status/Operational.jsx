import React, {Component} from 'react'
import PropTypes from 'prop-types';

class Operational extends Component {
    render() {
        const {operational} = this.props;
        return (
            <li className="list-group-item">
            <span class="badge"><span class="glyphicon glyphicon-ok" aria-hidden="true"></span></span>
                {operational.url}
            </li>
        )
    }
}

Operational.propTypes = {
    operational: PropTypes.object.isRequired
}

export default Operational
