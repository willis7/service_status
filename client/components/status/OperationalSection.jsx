import React, {Component} from 'react';
import PropTypes from 'prop-types';
import OperationalList from './OperationalList.jsx';

class OperationalSection extends Component {
    render(){
        return(
            <div>
                <OperationalList {...this.props} />
            </div>
        )
    }
}

OperationalSection.propTypes = {
    operationals: PropTypes.array.isRequired
}

export default OperationalSection
